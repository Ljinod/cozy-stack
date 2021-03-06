package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/cozy/cozy-stack/client"
)

func writeFile(path string, cClient *client.Client, tw *tar.Writer, doc *client.DirOrFile) error {
	readCloser, err := cClient.DownloadByID(doc.ID)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	hdr := &tar.Header{
		Name:       path,
		Mode:       0644,
		Size:       doc.Attrs.Size,
		ModTime:    doc.Attrs.CreatedAt,
		AccessTime: doc.Attrs.CreatedAt,
		ChangeTime: doc.Attrs.UpdatedAt,
	}
	if doc.Attrs.Executable {
		hdr.Mode = 0755
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := io.Copy(tw, readCloser); err != nil {
		return err
	}
	return nil
}

func export(tw *tar.Writer, cClient *client.Client) error {

	root := "/Documents"

	err := cClient.WalkByPath(root, func(path string, doc *client.DirOrFile, err error) error {

		if doc.Attrs.Type == client.DirType {
			fmt.Println("directory")
		} else if doc.Attrs.Type == client.FileType {
			if err := writeFile(path, cClient, tw, doc); err != nil {
				return err
			}
		} else {
			fmt.Println("type not found")
		}

		return nil

	})

	return err
}

func tardir(w io.Writer, cClient *client.Client) error {
	//gzip writer
	gw := gzip.NewWriter(w)
	defer gw.Close()

	//tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	err := export(tw, cClient)

	return err

}
