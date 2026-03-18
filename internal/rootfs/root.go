/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rootfs

import (
	"io/fs"
	"path"
	unixpath "path"
)

type rootFS struct {
	fs   fs.FS
	root string
}

// Open opens the named file within the base directory of the rootFS.
// It joins the base directory and the name using Unix-style path separators,
// ensuring compatibility with io/fs.Open which uses slash-separated paths on all systems.
func (f rootFS) Open(name string) (fs.File, error) {
	cleaned := path.Clean(name)
	if unixpath.IsAbs(cleaned) || !fs.ValidPath(cleaned) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	path := unixpath.Join(f.root, cleaned)
	return f.fs.Open(path)
}

var _ fs.FS = (*rootFS)(nil)

func New(fs fs.FS, root string) fs.FS {
	return &rootFS{fs: fs, root: root}
}
