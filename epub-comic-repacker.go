package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

func shinkName(filename string) string {
	fbase := filepath.Base(filename)
	reg, err := regexp.Compile(`(.*moe\])(.*)(\.kepub)(\.epub)$`)
	if err != nil {
		panic("regexp works with error")
	}
	shinkname := reg.ReplaceAllString(fbase, "$2$4")
	return shinkname
}

func getMainName(filename string) string {
	fbase := filepath.Base(filename)
	fext := filepath.Ext(filename)
	fmain := strings.TrimSuffix(fbase, fext)
	return fmain
}

func findAttrValue(r io.Reader, attrname string) (value string) {
	tokenizer := html.NewTokenizer(r)
	for tokenizer.Token().Data != "html" {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			if tokenizer.Err() == io.EOF {
				return
			}
			fmt.Printf("Error: %v", tokenizer.Err())
			return
		}
		tagName, _ := tokenizer.TagName()
		//to find the first value of tag named "img"
		if string(tagName) == "img" {
			attrKey, attrValue, _ := tokenizer.TagAttr()
			if string(attrKey) == attrname {
				return string(attrValue)
			}
		}
	}
	return
}

func unZipFiles(sourcefile string, cachefolder string) ([]string, error) {
	var extractfiles, extractimgs []string
	var imgname, imgpath []string
	r, err := zip.OpenReader(sourcefile)
	if err != nil {
		return extractfiles, err
	}
	defer r.Close()
	for _, f := range r.File {
		// //extract source zipfile with full address
		// desfpath := filepath.Join(cachefolder, getMainName(sourcefile), f.Name)
		coversavepath := filepath.Join(cachefolder, getMainName(sourcefile), filepath.Base(f.Name))
		extractfiles = append(extractfiles, coversavepath)
		// //create directory
		// if f.FileInfo().IsDir() {
		// 	os.MkdirAll(desfpath, os.ModePerm)
		// 	continue
		// }
		if err = os.MkdirAll(filepath.Dir(coversavepath), os.ModePerm); err != nil {
			return extractfiles, err
		}

		//to get cover.jpg
		reg, err := regexp.Compile(`cover\.(jpg|png)$`)
		if err != nil {
			panic("regexp works with error")
		}
		if reg.MatchString(f.FileInfo().Name()) {
			outfile, err := os.OpenFile(coversavepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return extractfiles, err
			}
			//open file in source zipfile
			rc, err := f.Open()
			if err != nil {
				return extractfiles, err
			}
			_, err = io.Copy(outfile, rc)
			//not use defer to close the file
			outfile.Close()
			rc.Close()
			if err != nil {
				return extractfiles, err
			}
		}

		//to get the "src" info pased from html and prepare the image names
		reg, err = regexp.Compile(`\d\.(html)$`)
		if err != nil {
			log.Fatal("regexp works with error")
		}
		if reg.MatchString(f.FileInfo().Name()) {
			in := getMainName(f.FileInfo().Name()) + ".jpg"
			imgname = append(imgname, in)

			rc, err := f.Open()
			if err != nil {
				return extractfiles, err
			}
			//to get the tag named "src"
			tagvalue := findAttrValue(rc, "src")
			imgpath = append(imgpath, tagvalue)
			rc.Close()
			if err != nil {
				return extractfiles, err
			}
		}
	}

	//second loop to change the parsed image name to the name of html
	for _, f2 := range r.File {
		for i := 0; i < len(imgname); i++ {
			imgsavepath := filepath.Join(cachefolder, getMainName(sourcefile), imgname[i])
			extractimgs = append(extractimgs, imgsavepath)

			if f2.FileInfo().Name() == filepath.Base(imgpath[i]) {
				outfile, err := os.OpenFile(imgsavepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f2.Mode())
				if err != nil {
					return extractimgs, err
				}

				rc, err := f2.Open()
				if err != nil {
					return extractimgs, err
				}
				_, err = io.Copy(outfile, rc)

				outfile.Close()
				rc.Close()
				fmt.Println("Extract: " + imgsavepath)
				if err != nil {
					return extractimgs, err
				}
			}
		}
	}
	return extractfiles, nil
}

func getFilelist(folder string) ([]string, error) {
	var result []string
	filepath.Walk(folder, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Println(err.Error())
			return err
		}
		if !fi.IsDir() {
			//if want to ignore the directory, return filepath.SkipDir
			//return filepath.SkipDir
			result = append(result, path)
		}
		return nil
	})
	return result, nil
}

func zipFiles(desfile string, srcfiles []string, oldform, newform string) error {
	newZipFile, err := os.Create(desfile)
	if err != nil {
		return err
	}
	defer newZipFile.Close()
	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	//add files to zip
	for _, srcfile := range srcfiles {
		zipfile, err := os.Open(srcfile)
		if err != nil {
			return err
		}
		defer zipfile.Close()
		//get the info of file
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		//modify FileInforHeader() so to change any address we want
		header.Name = strings.Replace(srcfile, oldform, newform, -1)
		// optimize zip
		// more to reference http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err = io.Copy(writer, zipfile); err != nil {
			return err
		}
	}
	return nil
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

func main() {
	//Setup Ctrl+C handler
	setupCloseHandler()
	//create temporary directory "cache"
	const cachefolder string = "cache"
	var shunkzipfiles []string
	err := os.Mkdir(cachefolder, os.ModePerm)
	if os.IsExist(err) {
		log.Println("cachefolder already exists, ignore creating")
	}
	//create save directory "ouput"
	const outputfolder string = "output"
	err = os.Mkdir(outputfolder, os.ModePerm)
	if os.IsExist(err) {
		log.Println("outputfolder already exists, ignore creating")
	}

	//shink the name of .epub and change to .zip
	epubfiles, err := filepath.Glob("*.epub")
	if err != nil {
		log.Fatal("pattern error")
	}
	if epubfiles == nil {
		log.Fatal("no epub files found")
	}
	for _, efile := range epubfiles {
		zfile := getMainName(shinkName(efile)) + ".zip"
		os.Rename(efile, zfile)
		shunkzipfiles = append(shunkzipfiles, zfile)
	}

	//unzip .zip files
	for _, szfile := range shunkzipfiles {
		//ignore the file list of unzip file
		_, err := unZipFiles(szfile, cachefolder)
		if err != nil {
			log.Fatal(err)
		}
	}

	//get the directory name under cache folder
	folders, _ := ioutil.ReadDir(cachefolder)
	var foldenames []string
	for _, fo := range folders {
		foldenames = append(foldenames, fo.Name())
	}

	for _, foldename := range foldenames {
		storename := outputfolder + string(os.PathSeparator) + foldename + ".zip"
		sourcefolder := cachefolder + string(os.PathSeparator) + foldename
		filelist, err := getFilelist(sourcefolder)
		if err != nil {
			log.Fatalln("getAfllFiles gos wrong")
		}
		err = zipFiles(storename, filelist, sourcefolder+string(os.PathSeparator), "")
		if err != nil {
			log.Fatalln("zipFiles gos worng")
		}
		fmt.Printf("Zipfile: %s\n", storename)
	}

	//delete cachefolder
	err = os.RemoveAll(cachefolder)
	if err != nil {
		log.Println("delete cachefolder error")
	}
	//chagnge files extendtion back to .epub
	for _, szfiles := range shunkzipfiles {
		err := os.Rename(szfiles, getMainName(szfiles)+".epub")
		if err != nil {
			log.Println("rename shunkzipfiles error")
		}
	}
	fmt.Println("All finished, quit in 3s later")
	time.Sleep(3 * time.Second)
}
