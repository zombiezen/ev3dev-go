package ev3dev

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// A sysfs directory of ev3dev devices.  They all follow the pattern of
// `<prefix><n>`, where `<n>` is monotonically increasing.
type deviceDir struct {
	path   string
	prefix string

	// The highest scanned device number or -1. Anything lower than n has
	// either been placed in skipped or has been returned.
	n int

	// Set of previously scanned devices. Always in sorted order.
	skipped []skipEntry
}

type skipEntry struct {
	i    int
	addr address
}

func newDeviceDir(path, prefix string) *deviceDir {
	return &deviceDir{
		path:   path,
		prefix: prefix,
		n:      -1,
	}
}

// findByAddress finds the first device that matches the given address.
func (d *deviceDir) findByAddress(addr address) (string, error) {
	// Key assumption: once a device is created, its address will never change.

	deviceNames, err := d.list()
	if err != nil {
		return "", fmt.Errorf("find device %q: %w", addr, err)
	}
	skipIndex := 0
	for _, dn := range deviceNames {
		// Check if device is in skip list. The skip list stores the
		// address, so no need to read from kernel.
		for skipIndex < len(d.skipped) && d.skipped[skipIndex].i < dn.n {
			skipIndex++
		}
		if skipIndex < len(d.skipped) && d.skipped[skipIndex].i == dn.n {
			if d.skipped[skipIndex].addr != addr {
				continue
			}
			d.skipped = append(d.skipped[:skipIndex], d.skipped[skipIndex+1:]...)
			return filepath.Join(d.path, dn.name), nil
		}

		// Ignore device if number is below high-water mark and not in skip list.
		if d.n >= 0 && dn.n <= d.n {
			continue
		}

		// Read device address.
		addrFile, err := os.Open(filepath.Join(d.path, dn.name, "address"))
		if err != nil {
			return "", fmt.Errorf("find device %q: %w", addr, err)
		}
		devAddr, err := readAttrAddr(addrFile)
		addrFile.Close()
		if err != nil {
			return "", fmt.Errorf("find device %q: %w", addr, err)
		}
		d.n = dn.n
		if devAddr == addr {
			return filepath.Join(d.path, dn.name), nil
		}
		d.skipped = append(d.skipped, skipEntry{dn.n, devAddr})
	}
	return "", fmt.Errorf("find device %q: not found", addr)
}

func (d *deviceDir) list() ([]deviceName, error) {
	dir, err := os.Open(d.path)
	if err != nil {
		return nil, fmt.Errorf("list %s devices: %w", d.prefix, err)
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, fmt.Errorf("list %s devices: %w", d.prefix, err)
	}
	var deviceNames []deviceName
	for _, name := range names {
		dn := parseDeviceName(name, d.prefix)
		if dn != (deviceName{}) {
			deviceNames = append(deviceNames, dn)
		}
	}
	sort.Slice(deviceNames, func(i, j int) bool {
		return deviceNames[i].n < deviceNames[j].n
	})
	return deviceNames, nil
}

type deviceName struct {
	name string
	n    int
}

func parseDeviceName(name, prefix string) deviceName {
	if !strings.HasPrefix(name, prefix) {
		return deviceName{}
	}
	n, err := strconv.Atoi(name[len(prefix):])
	if err != nil {
		return deviceName{}
	}
	return deviceName{name, n}
}

func (dn deviceName) String() string {
	return dn.name
}

// readAttrBytes reads the sysfs attribute into p.
func readAttrBytes(file io.ReaderAt, p []byte) (n int, err error) {
	// According to the sysfs docs, a new value will be triggered
	// when pread(2) has an offset of 0.
	n, err = file.ReadAt(p, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		name := attrName(file)
		if name == "" {
			return n, fmt.Errorf("read attribute: %w", err)
		}
		return n, fmt.Errorf("read attribute %s: %w", name, err)
	}
	if bytes.HasSuffix(p[:n], []byte{'\n'}) {
		n--
	}
	return n, nil
}

// readAttrAddr reads and parses an address attribute value.
func readAttrAddr(file io.ReaderAt) (address, error) {
	var a address
	n, err := readAttrBytes(file, a[:])
	if err != nil {
		return address{}, err
	}
	// Replicates newAddress in order to avoid a copy.
	if n > len(a)-1 {
		name := attrName(file)
		if name == "" {
			return address{}, fmt.Errorf("read attribute: address too long")
		}
		return address{}, fmt.Errorf("read attribute %s: address too long", name)
	}
	copy(a[1:], a[:n])
	a[0] = byte(n)
	for i := n + 1; i < len(a); i++ {
		a[i] = 0
	}
	return a, nil
}

// readAttrInt reads and parses an integer attribute value.
func readAttrInt(file io.ReaderAt, bits int) (int64, error) {
	var buf [24]byte
	n, err := readAttrBytes(file, buf[:])
	if err != nil {
		return 0, err
	}
	i, err := strconv.ParseInt(string(buf[:n]), 10, bits)
	if err != nil {
		name := attrName(file)
		if name == "" {
			return 0, fmt.Errorf("read attribute: %w", err)
		}
		return 0, fmt.Errorf("read attribute %s: %w", name, err)
	}
	return i, nil
}

func openAttrWrite(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
}

// writeAttr writes a sysfs attribute value.
func writeAttr(file *os.File, p []byte) error {
	// Needed for fakes.
	if err := file.Truncate(0); err != nil {
		name := attrName(file)
		if name == "" {
			return fmt.Errorf("write attribute: %w", err)
		}
		return fmt.Errorf("write attribute %s: %w", name, err)
	}
	// sysfs wants all data in a single write, so we need to customize the
	// interrupted behavior.
	for {
		n, err := unix.Write(int(file.Fd()), p)
		if err == nil {
			return nil
		}
		if n > 0 || !errors.Is(err, unix.EINTR) {
			name := attrName(file)
			if name == "" {
				return fmt.Errorf("write attribute: %w", err)
			}
			return fmt.Errorf("write attribute %s: %w", name, err)
		}
	}
}

// writeAttrInt writes an integer sysfs attribute value.
func writeAttrInt(file *os.File, value int64) error {
	buf := strconv.AppendInt(make([]byte, 0, 24), value, 10)
	return writeAttr(file, buf)
}

// address is a port address string, like "spi0.1:S3". The zero value is
// the empty address.
type address [64]byte

func newAddress(s string) (address, error) {
	var a address
	if len(s) > len(a)-1 {
		return address{}, errors.New("address too long")
	}
	a[0] = byte(len(s))
	copy(a[1:], s)
	return a, nil
}

func (a address) String() string {
	return string(a[1 : a[0]+1])
}

func attrName(f interface{}) string {
	namer, ok := f.(interface{ Name() string })
	if !ok {
		return ""
	}
	return filepath.Base(namer.Name())
}
