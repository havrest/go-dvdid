package dvdid

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    "strings"
    "sort"
    "encoding/binary"
    "github.com/havrest/go-dvdid/dvdid/internal/dvdcrc64"
)

const dvdMaxReadSize int64 = 0x10000

type HashFileInfos struct {
    CreationTime uint64
    Size int64
    Name string
}

type HashFileBuf struct {
    Data []byte
    Size int64
}

type HashInfos struct {
    FilesList []HashFileInfos
    FilesBuf []HashFileBuf
}

func timeToFiletime(ftime time.Time) uint64 {
    return uint64(ftime.UnixNano()/100+11644473600*10000000)
}

func ComputeDVDId(discPath string) (string, error) {

    /* Check that the VIDEO_TS folder exists */
    fileInfos, err := os.Stat(filepath.Join(discPath, "VIDEO_TS"))
    if os.IsNotExist(err) || !fileInfos.IsDir() {
        return "", fmt.Errorf("\"VIDEO_TS\" could not be found in \"%s\" or is not a directory.", discPath)
    }

    /* Hash structure that will hold all information necessary to compute the CRC64 */
    var hashInfos HashInfos
    discId := ""

    /* List files in "VIDEO_TS" and retrieve necessary information from files */
        /* Open directory */
        var videoDir *os.File
        videoDir, err = os.Open(filepath.Join(discPath, "VIDEO_TS"))
        if err != nil {
            return "", fmt.Errorf("Failed to open directory \"%s\" to list files. %e", filepath.Join(discPath, "VIDEO_TS"), err)
        }
        defer videoDir.Close()

        /* Retrieve list of files */
        var files []os.FileInfo;
        files, err = videoDir.Readdir(-1)
        if err != nil {
            return "", fmt.Errorf("Failed to list files in directory \"%s\". %e", filepath.Join(discPath, "VIDEO_TS"), err)
        }

        /* Register infos */
        for _, f := range files {
            if !f.IsDir() {
                hashInfos.FilesList = append(hashInfos.FilesList, HashFileInfos{
                    CreationTime: timeToFiletime(f.ModTime()),
                    Size: f.Size(),
                    Name: strings.ToUpper(f.Name()),
                })
            }
        }
        sort.Slice(hashInfos.FilesList, func(i, j int) bool {
            return strings.Compare(hashInfos.FilesList[i].Name, hashInfos.FilesList[j].Name) < 0
        })

    /* Retrieve content of files "VIDEO_TS.IFO" and "VTS_01_0.IFO" */
    for _, fName := range []string{"VIDEO_TS.IFO", "VTS_01_0.IFO"} {
        /* Check that the file exists and retrieve file size */
        fileInfos, err := os.Stat(filepath.Join(discPath, "VIDEO_TS", fName))
        if os.IsNotExist(err) {
            return "", fmt.Errorf("\"%s\" could not be found in \"%s\" or is not a directory.", filepath.Join("VIDEO_TS", fName), discPath)
        }

        /* Set the number of bytes to read up to dvdMaxReadSize bytes */
        readSize := fileInfos.Size()
        if readSize > dvdMaxReadSize {
            readSize = dvdMaxReadSize
        }

        fBuf := HashFileBuf{
            Size: readSize,
            Data: make([]byte, readSize),
        }

        /* Open the file */
        var file *os.File
        file, err = os.Open(filepath.Join(discPath, "VIDEO_TS", fName))
        if err != nil {
            return "", fmt.Errorf("\"%s\" could not be opened in \"%s\" or is not a directory.", filepath.Join("VIDEO_TS", fName), discPath)
        }
        defer file.Close()

        /* Read the content of the file */
        var count int
        count, err = file.Read(fBuf.Data)
        if err != nil || count != int(readSize) {
        	return "", fmt.Errorf("\"%s\" could not be read in \"%s\" or is not a directory.", filepath.Join("VIDEO_TS", fName), discPath)
        }

        hashInfos.FilesBuf = append(hashInfos.FilesBuf, fBuf)
    }

    /* Compute the CRC64 */
    discIdCrc64 := dvdcrc64.New(dvdcrc64.MakeTable(dvdcrc64.DVD))
    b := make([]byte, 8)

        /* Files infos are added to the CRC */
        for _, fInfos := range hashInfos.FilesList {
            b = make([]byte, 8)
            binary.LittleEndian.PutUint64(b, fInfos.CreationTime)
            discIdCrc64.Write(b)

            b = make([]byte, 4)
            binary.LittleEndian.PutUint32(b, uint32(fInfos.Size))
            discIdCrc64.Write(b)

            discIdCrc64.Write([]byte(fInfos.Name))
            discIdCrc64.Write([]byte("\x00"))
        }

        /* Files content are added to the CRC */
        for _, fBuf := range hashInfos.FilesBuf {
            discIdCrc64.Write([]byte(fBuf.Data))
        }

    discId = fmt.Sprintf("%X-%X", discIdCrc64.Sum(nil)[0:4], discIdCrc64.Sum(nil)[4:])

    return discId, nil
}
