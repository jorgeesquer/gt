package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/gtlang/filesystem"
	"github.com/gtlang/gt/core"
)

type FileSystemObj struct {
	FS filesystem.FS
}

func NewFileSystem(fs filesystem.FS) *FileSystemObj {
	return &FileSystemObj{fs}
}

func (f *FileSystemObj) Type() string {
	return "os.FileSystem"
}

func (f *FileSystemObj) GetProperty(key string, vm *core.VM) (core.Value, error) {
	if f == nil {
		return core.UndefinedValue, nil
	}

	switch key {
	case "workingDir":
		s, err := f.FS.Getwd()
		if err != nil {
			return core.NullValue, err
		}
		return core.NewString(s), nil
	}
	return core.UndefinedValue, nil
}

func (f *FileSystemObj) GetMethod(name string) core.NativeMethod {
	if f == nil {
		return nil
	}

	switch name {
	case "stat":
		return f.stat
	case "readAll":
		return f.readAll
	case "readString":
		return f.readString
	case "readAllIfExists":
		return f.readAllIfExists
	case "readStringIfExists":
		return f.readStringIfExists
	case "write":
		return f.write
	case "append":
		return f.append
	case "mkdir":
		return f.mkdir
	case "readDir":
		return f.readDir
	case "readNames":
		return f.readNames
	case "exists":
		return f.exists
	case "rename":
		return f.rename
	case "removeAll":
		return f.removeAll
	case "abs":
		return f.abs
	case "chdir":
		return f.chdir
	case "open":
		return f.open
	case "openIfExists":
		return f.openIfExists
	case "openForWrite":
		return f.openForWrite
	case "openForAppend":
		return f.openForAppend
	}
	return nil
}

func (f *FileSystemObj) rename(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	source := args[0].ToString()
	dest := args[1].ToString()

	if err := f.FS.Rename(source, dest); err != nil {
		if os.IsNotExist(err) {
			return core.NullValue, fmt.Errorf("rename %v to %v: %w", source, dest, err)
		}
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (f *FileSystemObj) removeAll(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	if err := f.FS.RemoveAll(name); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (f *FileSystemObj) openIfExists(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	fi, err := f.FS.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	return core.NewObject(newFile(fi, vm)), nil
}

func (f *FileSystemObj) open(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	fi, err := f.FS.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return core.NullValue, fmt.Errorf("open %v: %w", name, err)
		}
		return core.NullValue, err
	}

	return core.NewObject(newFile(fi, vm)), nil
}

func (f *FileSystemObj) openForWrite(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	fi, err := f.FS.OpenForWrite(name)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(newFile(fi, vm)), nil
}

func (f *FileSystemObj) openForAppend(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	fi, err := f.FS.OpenForAppend(name)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(newFile(fi, vm)), nil
}

func newFile(fi filesystem.File, vm *core.VM) *file {
	f := &file{f: fi}
	vm.SetGlobalFinalizer(f)
	return f
}

type file struct {
	io.ReaderAt
	f      filesystem.File
	closed bool
}

func (f *file) Type() string {
	return "os.File"
}

func (f *file) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	return f.f.Close()
}

func (f *file) Write(p []byte) (n int, err error) {
	return f.f.Write(p)
}

func (f *file) Read(p []byte) (n int, err error) {
	return f.f.Read(p)
}

func (f *file) ReadAt(p []byte, off int64) (n int, err error) {
	return f.f.ReadAt(p, off)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	return f.f.Seek(offset, whence)
}

func (f *file) Stat() (os.FileInfo, error) {
	return f.f.Stat()
}

func (f *file) GetMethod(name string) core.NativeMethod {
	switch name {
	case "write":
		return f.write
	case "writeAt":
		return f.writeAt
	case "read":
		return f.read
	case "close":
		return f.close
	}
	return nil
}

func (f *file) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 parameter")
	}

	a := args[0]

	if err := Write(f.f, a, vm); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (f *file) writeAt(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 2 {
		return core.NullValue, fmt.Errorf("expected 2 parameters")
	}

	a := args[0]

	offV := args[1]
	if offV.Type != core.Int {
		return core.NullValue, fmt.Errorf("expected parameter 2 to be int")
	}

	if err := WriteAt(f.f, a, offV.ToInt(), vm); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (f *file) read(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Bytes); err != nil {
		return core.NullValue, err
	}

	buf := args[0].ToBytes()

	n, err := f.f.Read(buf)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewInt(n), nil
}

func (f *file) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("no parameters expected")
	}
	if !f.closed {
		f.closed = true
		f.f.Close()
	}
	return core.NullValue, nil
}

func (f *FileSystemObj) abs(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	path := args[0].ToString()
	abs, err := f.FS.Abs(path)
	if err != nil {
		if os.IsNotExist(err) {
			return core.NullValue, fmt.Errorf("open %v: %w", path, err)
		}
		return core.NullValue, err
	}

	return core.NewString(abs), nil
}

func (f *FileSystemObj) chdir(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	dir := args[0].ToString()
	if err := f.FS.Chdir(dir); err != nil {
		if os.IsNotExist(err) {
			return core.NullValue, fmt.Errorf("open %v: %w", dir, err)
		}
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (f *FileSystemObj) exists(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	if _, err := f.FS.Stat(name); err != nil {
		return core.FalseValue, nil
	}
	return core.TrueValue, nil
}

func (f *FileSystemObj) readNames(args []core.Value, vm *core.VM) (core.Value, error) {
	var name string
	var recursive bool
	l := len(args)

	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got %d", len(args))
	}

	if l > 2 {
		return core.NullValue, fmt.Errorf("expected 2 arguments max, got %d", len(args))
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].TypeName())
	}

	name = args[0].ToString()

	if l == 2 {
		if args[1].Type != core.Bool {
			return core.NullValue, fmt.Errorf("expected argument 2 to be a boolean, got %v", args[0].TypeName())
		}
		recursive = args[1].ToBool()
	}

	fis, err := ReadNames(f.FS, name, recursive)
	if err != nil {
		return core.NullValue, err
	}

	result := make([]core.Value, len(fis))

	for i, fi := range fis {
		result[i] = core.NewString(fi)
	}

	return core.NewArrayValues(result), nil
}

// ReadNames reads the directory and file names contained in dirname.
func ReadNames(fs filesystem.FS, dirname string, recursive bool) ([]string, error) {
	n, err := readNames(fs, dirname, true, recursive)
	if err != nil {
		return nil, err
	}
	sort.Strings(n)
	return n, nil
}

func readNames(fs filesystem.FS, dirname string, removeTopDir, recursive bool) ([]string, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("open %v: %w", dirname, err)
		}
		return nil, err
	}

	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	var names []string

	for _, l := range list {
		name := filepath.Join(dirname, l.Name())
		names = append(names, name)

		if recursive && l.IsDir() {
			sub, err := readNames(fs, name, false, true)
			if err != nil {
				return nil, err
			}

			//			if removeTopDir {
			//				for i, v := range sub {
			//					j := strings.IndexRune(v, os.PathSeparator) + 1
			//					sub[i] = v[j:]
			//				}
			//			}

			names = append(names, sub...)
		}
	}

	return names, nil
}

func (f *FileSystemObj) stat(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	fi, err := f.FS.Stat(name)
	if err != nil {
		// ignore errors. Just return null if is invalid
		return core.NullValue, nil
	}

	return core.NewObject(fileInfo{fi}), nil
}

func (f *FileSystemObj) readDir(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	file, err := f.FS.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return core.NullValue, fmt.Errorf("open %v: %w", name, err)
		}
		return core.NullValue, err
	}

	list, err := file.Readdir(-1)
	file.Close()
	if err != nil {
		return core.NullValue, err
	}

	result := make([]core.Value, len(list))

	for i, fi := range list {
		result[i] = core.NewObject(fileInfo{fi})
	}

	return core.NewArrayValues(result), nil
}

func (f *FileSystemObj) mkdir(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	if err := f.FS.MkdirAll(name); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (f *FileSystemObj) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 2 {
		return core.NullValue, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument 1 to be a string, got %d", args[0].Type)
	}
	name := args[0].ToString()

	file, err := f.FS.OpenForWrite(name)
	if err != nil {
		return core.NullValue, err
	}

	if err := Write(file, args[1], vm); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (f *FileSystemObj) append(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 2 {
		return core.NullValue, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument 1 to be a string, got %s", args[0].TypeName())
	}
	name := args[0].ToString()

	b := args[1]
	switch b.Type {
	case core.Bytes, core.String:
	default:
		return core.NullValue, fmt.Errorf("expected argument 2 to be a string or byte array, got %s", args[1].TypeName())
	}

	f.FS.AppendPath(name, b.ToBytes())
	return core.NullValue, nil
}

func (f *FileSystemObj) readAll(args []core.Value, vm *core.VM) (core.Value, error) {
	b, err := f.read(false, args, vm)
	if err != nil {
		return core.NullValue, err
	}
	if b == nil {
		return core.NullValue, nil
	}
	return core.NewBytes(b), nil
}

func (f *FileSystemObj) readString(args []core.Value, vm *core.VM) (core.Value, error) {
	b, err := f.read(false, args, vm)
	if err != nil {
		return core.NullValue, err
	}
	if b == nil {
		return core.NullValue, nil
	}
	return core.NewString(string(b)), nil
}

func (f *FileSystemObj) readAllIfExists(args []core.Value, vm *core.VM) (core.Value, error) {
	b, err := f.read(true, args, vm)
	if err != nil {
		return core.NullValue, err
	}
	if b == nil {
		return core.NullValue, nil
	}
	return core.NewBytes(b), nil
}

func (f *FileSystemObj) readStringIfExists(args []core.Value, vm *core.VM) (core.Value, error) {
	b, err := f.read(true, args, vm)
	if err != nil {
		return core.NullValue, err
	}
	if b == nil {
		return core.NullValue, nil
	}
	return core.NewString(string(b)), nil
}

func (f *FileSystemObj) read(ifExists bool, args []core.Value, vm *core.VM) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
	}
	if args[0].Type != core.String {
		return nil, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}
	name := args[0].ToString()

	file, err := f.FS.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			if ifExists {
				return nil, nil
			}
			return nil, fmt.Errorf("open %v: %w", name, err)
		}
		return nil, err
	}
	b, err := ioutil.ReadAll(file)
	file.Close()
	if err != nil {
		return nil, err
	}
	return b, nil
}

type fileInfo struct {
	fi os.FileInfo
}

func (f fileInfo) Type() string {
	return "os.FileInfo"
}

func (f fileInfo) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "name":
		return core.NewString(f.fi.Name()), nil
	case "modTime":
		return core.NewObject(TimeObj(f.fi.ModTime())), nil
	case "isDir":
		return core.NewBool(f.fi.IsDir()), nil
	case "size":
		return core.NewInt64(f.fi.Size()), nil
	}
	return core.UndefinedValue, nil
}
