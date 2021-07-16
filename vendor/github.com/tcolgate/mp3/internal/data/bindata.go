package data

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unsafe"
	"os"
	"time"
	"io/ioutil"
	"path"
	"path/filepath"
)

func bindata_read(data, name string) ([]byte, error) {
	var empty [0]byte
	sx := (*reflect.StringHeader)(unsafe.Pointer(&data))
	b := empty[:]
	bx := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bx.Data = sx.Data
	bx.Len = len(data)
	bx.Cap = bx.Len

	gz, err := gzip.NewReader(bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindata_file_info struct {
	name string
	size int64
	mode os.FileMode
	modTime time.Time
}

func (fi bindata_file_info) Name() string {
	return fi.name
}
func (fi bindata_file_info) Size() int64 {
	return fi.size
}
func (fi bindata_file_info) Mode() os.FileMode {
	return fi.mode
}
func (fi bindata_file_info) ModTime() time.Time {
	return fi.modTime
}
func (fi bindata_file_info) IsDir() bool {
	return false
}
func (fi bindata_file_info) Sys() interface{} {
	return nil
}

var _silent_1frame_go = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\x64\x90\x41\x4b\x3b\x31\x10\xc5\xcf\x3b\x9f\xe2\xfd\xf7\xb4\x85\x7f\x1b\xaa\x17\x11\x7a\x50\xc1\x8b\x47\x8f\x22\x92\xee\xce\xa6\xa1\x9b\x99\x90\xa4\x4a\x91\x7e\x77\x77\xe3\x45\xf1\x10\x02\x33\x6f\x7e\xef\xcd\x44\xdb\x1f\xad\x63\x0c\xb6\x58\x22\x1f\xa2\xa6\x82\x76\x52\xd7\x12\xbd\xdb\x84\x8e\x1a\x63\xf0\xec\x27\x96\x72\x7f\x2e\x9c\xe1\x33\xca\x81\x91\xec\x47\x1d\xc2\x98\x34\xc0\x62\x7b\x75\x83\xa7\xbd\xc9\xd0\x11\x93\x0d\x0c\x96\x5e\x07\x1e\x20\x5a\x0e\x5e\x9c\x70\xce\xd4\xfc\x04\xbd\xbc\xee\xe7\x9f\x56\x44\xc6\x38\xbd\x75\x2c\x9c\x6c\x61\x38\x5d\xef\xbd\x54\xf6\x3a\x1e\xdd\xb7\xcb\x5a\x34\x70\xe8\x35\x9e\xb1\x31\x34\x9e\xa4\x87\x17\x5f\xba\x15\x3e\xa9\x59\x82\x72\xaa\x4f\xd3\x2f\x93\xff\xb5\xbe\xc3\x5d\xce\x5c\xba\x76\x41\x99\x5c\xdb\x6f\xdb\x31\xcd\x31\x37\x21\x5e\xb7\x2b\x6a\xfc\x58\x95\xff\x76\x10\x3f\x2d\xcc\x66\xbe\xc1\xe6\x71\xd6\x4f\x63\xd7\x3e\xe8\x69\xaa\x9b\x40\x23\x0b\xfe\x10\x60\x17\xfe\xc2\xb9\xd0\x85\xbe\x02\x00\x00\xff\xff\x0e\x9e\x37\x70\x54\x01\x00\x00"

func silent_1frame_go_bytes() ([]byte, error) {
	return bindata_read(
		_silent_1frame_go,
		"silent_1frame.go",
	)
}

func silent_1frame_go() (*asset, error) {
	bytes, err := silent_1frame_go_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "silent_1frame.go", size: 340, mode: os.FileMode(420), modTime: time.Unix(1424984811, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

var _silent_1frame_mp3 = "\x1f\x8b\x08\x00\x00\x09\x6e\x88\x00\xff\xfa\xff\x7b\x43\x0a\x03\xff\x07\x06\x86\x4c\x06\x06\x06\x0e\x06\x06\x5e\x05\x06\x06\x46\x20\x5a\x02\xe4\x02\x99\x26\x0d\x0c\x0c\x2c\x3e\x8e\xbe\xae\xc6\x7a\x96\x96\x7a\xa6\x0c\xa3\x60\x14\x50\x08\x00\x01\x00\x00\xff\xff\xa1\x6f\x84\x53\x72\x02\x00\x00"

func silent_1frame_mp3_bytes() ([]byte, error) {
	return bindata_read(
		_silent_1frame_mp3,
		"silent_1frame.mp3",
	)
}

func silent_1frame_mp3() (*asset, error) {
	bytes, err := silent_1frame_mp3_bytes()
	if err != nil {
		return nil, err
	}

	info := bindata_file_info{name: "silent_1frame.mp3", size: 626, mode: os.FileMode(420), modTime: time.Unix(1424763406, 0)}
	a := &asset{bytes: bytes, info:  info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"silent_1frame.go": silent_1frame_go,
	"silent_1frame.mp3": silent_1frame_mp3,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() (*asset, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"silent_1frame.go": &_bintree_t{silent_1frame_go, map[string]*_bintree_t{
	}},
	"silent_1frame.mp3": &_bintree_t{silent_1frame_mp3, map[string]*_bintree_t{
	}},
}}

// Restore an asset under the given directory
func RestoreAsset(dir, name string) error {
        data, err := Asset(name)
        if err != nil {
                return err
        }
        info, err := AssetInfo(name)
        if err != nil {
                return err
        }
        err = os.MkdirAll(_filePath(dir, path.Dir(name)), os.FileMode(0755))
        if err != nil {
                return err
        }
        err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
        if err != nil {
                return err
        }
        err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
        if err != nil {
                return err
        }
        return nil
}

// Restore assets under the given directory recursively
func RestoreAssets(dir, name string) error {
        children, err := AssetDir(name)
        if err != nil { // File
                return RestoreAsset(dir, name)
        } else { // Dir
                for _, child := range children {
                        err = RestoreAssets(dir, path.Join(name, child))
                        if err != nil {
                                return err
                        }
                }
        }
        return nil
}

func _filePath(dir, name string) string {
        cannonicalName := strings.Replace(name, "\\", "/", -1)
        return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

