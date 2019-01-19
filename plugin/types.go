package plugin

import (
	"context"
	"io"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// ==== Wash Protocols and Resources ====

// Entry in a filesystem
type Entry struct {
	Client interface{}
	Name   string
	Isdir  bool
}

// IFileBuffer represents a file that can be ReadAt and Close.
type IFileBuffer interface {
	io.ReaderAt
}

// Attributes of resources.
type Attributes struct {
	Mtime time.Time
	Size  uint64
}

// GroupTraversal that plugins are expected to model.
type GroupTraversal interface {
	Find(ctx context.Context, name string) (*Entry, error)
	List(ctx context.Context) ([]Entry, error)
}

// Content protocol.
type Content interface {
	Open(ctx context.Context, name string) (IFileBuffer, error)
}

// Stream protocol for data that we only stream?

// Metadata covers protocols supported by all resources.
type Metadata interface {
	Attr(ctx context.Context, name string) (*Attributes, error)
	Xattr(ctx context.Context, name string) (map[string][]byte, error)
}

// DirProtocol is protocols expected of a Directory resource.
type DirProtocol interface {
	GroupTraversal
	Metadata
}

// FileProtocol is protocols expected of a File resource.
type FileProtocol interface {
	GroupTraversal
	Content
	Metadata
}

// ==== FUSE Adapters ====

// Node represents a filesystem node
type Node = fs.Node

// ENOENT states the entity does not exist
const (
	ENOENT  = fuse.ENOENT
	ENOTSUP = fuse.ENOTSUP
)

// FS contains the core filesystem data.
type FS struct {
	Clients map[string]DirProtocol
}

var _ fs.FS = (*FS)(nil)

// Dir represents a directory within the system, with the client
// necessary to represent it and the full path to the directory.
type Dir struct {
	client DirProtocol
	name   string
}

var _ fs.Node = (*Dir)(nil)
var _ = fs.NodeRequestLookuper(&Dir{})
var _ = fs.HandleReadDirAller(&Dir{})

// File contains metadata about the file.
type File struct {
	client FileProtocol
	name   string
}

var _ fs.Node = (*File)(nil)
var _ = fs.NodeOpener(&File{})
var _ = fs.NodeGetxattrer(&File{})
var _ = fs.NodeListxattrer(&File{})

// FileHandle contains an IO object that can be read.
type FileHandle struct {
	r IFileBuffer
}

var _ fs.Handle = (*FileHandle)(nil)
var _ = fs.HandleReleaser(&FileHandle{})
var _ = fs.HandleReader(&FileHandle{})
