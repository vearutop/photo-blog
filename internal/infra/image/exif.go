package image

import (
	"bytes"
	"fmt"
	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"os"
)

type ExifReader struct {
}

//type IfdEntry struct {
//	IfdPath     string                      `json:"ifd_path"`
//	FqIfdPath   string                      `json:"fq_ifd_path"`
//	IfdIndex    int                         `json:"ifd_index"`
//	TagId       uint16                      `json:"tag_id"`
//	TagName     string                      `json:"tag_name"`
//	TagTypeId   exifcommon.TagTypePrimitive `json:"tag_type_id"`
//	TagTypeName string                      `json:"tag_type_name"`
//	UnitCount   uint32                      `json:"unit_count"`
//	Value       interface{}                 `json:"value"`
//	ValueString string                      `json:"value_string"`
//}
//
//func (er *ExifReader) Foo() error {
//	im, err := exifcommon.NewIfdMappingWithStandard()
//	if err != nil {
//		return err
//	}
//
//	ti := exif.NewTagIndex()
//
//	entries := make([]IfdEntry, 0)
//	visitor := func(fqIfdPath string, ifdIndex int, tagId uint16, tagType exifcommon.TagTypePrimitive, valueContext exifcommon.ValueContext) (err error) {
//		ifdPath, err := im.StripPathPhraseIndices(fqIfdPath)
//
//		if err != nil {
//			return err
//		}
//
//		it, err := ti.Get(ifdPath, tagId)
//		if err != nil {
//			if log.Is(err, exif.ErrTagNotFound) {
//				fmt.Printf("WARNING: Unknown tag: [%s] (%04x)\n", ifdPath, tagId)
//				return nil
//			} else {
//				log.Panic(err)
//			}
//		}
//
//		valueString := ""
//		var value interface{}
//		if tagType.Type() == exif.TypeUndefined {
//			var err error
//			value, err = valueContext.Undefined()
//			if err != nil {
//				if err == exif.ErrUnhandledUnknownTypedTag {
//					value = nil
//				} else {
//					log.Panic(err)
//				}
//			}
//
//			valueString = fmt.Sprintf("%v", value)
//		} else {
//			valueString, err = valueContext.FormatFirst()
//			log.PanicIf(err)
//
//			value = valueString
//		}
//
//		entry := IfdEntry{
//			IfdPath:     ifdPath,
//			FqIfdPath:   fqIfdPath,
//			IfdIndex:    ifdIndex,
//			TagId:       tagId,
//			TagName:     it.Name,
//			TagTypeId:   tagType.Type(),
//			TagTypeName: tagType.Name(),
//			UnitCount:   valueContext.UnitCount(),
//			Value:       value,
//			ValueString: valueString,
//		}
//
//		entries = append(entries, entry)
//
//		return nil
//	}
//
//	_, err = exif.Visit(exifcommon.IfdStandardIfdIdentity, im, ti, rawExif, visitor)
//
//}

func (er *ExifReader) Read(filepath string) error {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	i := bytes.Index(b, []byte(`Rating`))
	println(string(b[i : i+10000]))

	rawExif, err := exif.SearchFileAndExtractExif(filepath)
	if err != nil {
		return err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return err
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return err
	}

	ifd, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdGpsInfoStandardIfdIdentity)
	if err != nil {
		return err
	}

	gi, err := ifd.GpsInfo()
	if err != nil {
		return err
	}

	cb := func(ifd *exif.Ifd, ite *exif.IfdTagEntry) error {
		//println(ifd.String(), ite.String())
		v, _ := ite.Value()
		s, _ := ite.Format()
		println(ite.IfdPath(), ite.TagName(), s)
		fmt.Printf("%#v\n", v)
		// Something useful.

		return nil
	}

	err = index.RootIfd.EnumerateTagsRecursively(cb)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n%#v \n", gi.String(), gi)

	return nil
}
