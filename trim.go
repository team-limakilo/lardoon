package lardoon

import (
	"io"

	"github.com/b1naryth1ef/jambon"
	"github.com/b1naryth1ef/jambon/tacview"
)

func trimTacView(path string, writer io.Writer, start, end int) error {
	file, err := jambon.OpenReadableTacView(path)
	if err != nil {
		return err
	}

	reader, err := tacview.NewParser(file)
	if err != nil {
		return err
	}

	tacWriter := tacview.NewRawWriter(writer)

	err = tacview.TrimRaw(reader, tacWriter, float64(start), float64(end))
	if err != nil {
		return err

	}
	return nil
}
