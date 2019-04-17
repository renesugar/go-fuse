// Copyright 2019 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nodefs_test

import (
	"context"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/nodefs"
)

// files contains the files we will expose as a file system
var files = map[string]string{
	"file":              "content",
	"subdir/other-file": "other-content",
}

// inMemoryFS is the root of the tree
type inMemoryFS struct {
	nodefs.Inode
}

// Ensure that we implement NodeOnAdder
var _ = (nodefs.NodeOnAdder)((*inMemoryFS)(nil))

// OnAdd is called on mounting the file system. Use it to populate
// the file system tree.
func (root *inMemoryFS) OnAdd(ctx context.Context) {
	for name, content := range files {
		dir, base := filepath.Split(name)

		p := &root.Inode

		// Add directories leading up to the file.
		for _, component := range strings.Split(dir, "/") {
			if len(component) == 0 {
				continue
			}
			ch := p.GetChild(component)
			if ch == nil {
				// Create a directory
				ch = p.NewPersistentInode(ctx, &nodefs.Inode{},
					nodefs.StableAttr{Mode: syscall.S_IFDIR})
				// Add it
				p.AddChild(component, ch, true)
			}

			p = ch
		}

		// Create the file. The Inode must be persistent,
		// because its life time is not under control of the
		// kernel.
		child := p.NewPersistentInode(ctx, &nodefs.MemRegularFile{
			Data: []byte(content),
		}, nodefs.StableAttr{})

		// And add it
		p.AddChild(base, child, true)
	}
}

// This demonstrates how to build a file system in memory.
func Example() {
	// This is where we'll mount the FS
	mntDir, _ := ioutil.TempDir("", "")

	root := &inMemoryFS{}
	server, err := nodefs.Mount(mntDir, root, &nodefs.Options{
		MountOptions: fuse.MountOptions{Debug: true},

		// This adds read permissions to the files and
		// directories, which is necessary for doing a chdir
		// into the mount.
		DefaultPermissions: true,
	})
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Mounted on %s", mntDir)
	log.Printf("Unmount by calling 'fusermount -u %s'", mntDir)

	// Wait until unmount before exiting
	server.Wait()
}