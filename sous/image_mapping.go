package sous

import (
	"fmt"

	"github.com/opentable/sous/util/docker_registry"
)

type (
	NameCache struct {
		DockerNameLookup
		SourceNameLookup
	}

	SourceRecord struct {
		SourceLocation SourceLocation
		etag           string
	}

	ImageName string

	SourceNameLookup map[ImageName]SourceRecord
	DockerNameLookup map[SourceLocation]ImageName

	NoImageNameFound struct {
		SourceLocation
	}

	NoSourceLocationFound struct {
		ImageName
	}
)

var theNameCache = NameCache{
	make(DockerNameLookup),
	make(SourceNameLookup),
}

func (e NoImageNameFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.SourceLocation)
}

func (e NoSourceLocationFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.ImageName)
}

func (dl DockerNameLookup) GetImageName(sl SourceLocation) (ImageName, error) {
	if in, ok := dl[sl]; ok {
		return in, nil
	} else {
		return "", NoImageNameFound{sl}
	}
}

func (sn SourceNameLookup) getSourceRecord(in ImageName) (SourceRecord, error) {
	if sr, ok := sn[in]; ok {
		return sr, nil
	} else {
		return SourceRecord{}, NoSourceLocationFound{in}
	}
}

func (sn SourceNameLookup) GetSourceLocation(in ImageName) (SourceLocation, error) {
	if sr, ok := sn[in]; ok {
		return sr.SourceLocation, nil
	} else {
		return SourceLocation{}, NoSourceLocationFound{in}
	}
}

func (dl DockerNameLookup) InsertDockerName(sl SourceLocation, in ImageName) error {
	dl[sl] = in
	return nil
}

func (sn SourceNameLookup) InsertSourceLocation(in ImageName, sl SourceLocation, etag string) error {
	sn[in] = SourceRecord{sl, etag}
	return nil
}

func (nc *NameCache) Insert(sl SourceLocation, in ImageName, etag string) error {
	err := nc.InsertSourceLocation(in, sl, etag)
	if err != nil {
		return err
	}

	err = nc.InsertDockerName(sl, in)
	if err != nil {
		return err
	}

	return nil
}

func GetSourceLocation(dr docker_registry.Client, in ImageName) (SourceLocation, error) {
	sr, err := theNameCache.getSourceRecord(in)

	newSL, etag, err := retreiveSourceLocation(in, sr.etag)
	if exists, ok := err.(NotModifiedErr); ok {
		return sr.SourceLocation, nil
	}
	if err != nil {
		return SourceLocation{}, err
	}

	if sr.SourceLocation != newSL {
		theNameCache.Insert(newSL, in, etag)
	}
	return newSL, nil
}
