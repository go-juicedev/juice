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

package juice

import (
	"io/fs"
	unixpath "path"
)

type fsRoot struct {
	fs      fs.FS
	baseDir string
}

// Open opens the named file within the base directory of the fsRoot.
// It joins the base directory and the name using Unix-style path separators,
// ensuring compatibility with io/fs.Open which uses slash-separated paths on all systems.
func (f fsRoot) Open(name string) (fs.File, error) {
	path := unixpath.Join(f.baseDir, name)
	return f.fs.Open(path)
}

var _ fs.FS = (*fsRoot)(nil)

func newFsRoot(fs fs.FS, basedir string) fs.FS {
	return &fsRoot{fs: fs, baseDir: basedir}
}
