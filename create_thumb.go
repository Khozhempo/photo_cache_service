package main

import (
	"errors"
	//	"fmt"
	"image"
	//	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/disintegration/imaging"
	"github.com/kpango/glg"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

func thumbGoOver(thumbType int) bool { // 0 - thumb, 1 - image thumb
	glg.Infof("[thumbGoOver] %d files to operate", len(list2CreateThumb))
	var glgFunc string // тип раздела для log
	if thumbType == 0 {
		glgFunc = "TH"
	} else {
		glgFunc = "iTH"
	}
	for k, v := range list2CreateThumb {
		for len(list2CreateThumbPriority) > 0 { // на паузу, если есть задание на создание приоритетного списка
			time.Sleep(1 * time.Second)
		}

		startTime := time.Now().Round(time.Second) // time check performance
		//		glg.Debugf("[%s] processing: %s", glgFunc, v)
		list2CreateThumb, _ = removeSlice(list2CreateThumb, k) // удаляем запись от списка
		if thumbType == 0 {                                    // если thumb creation
			if checkIfThumbInCache(v) || !checkIfImg(v) { // если файл существует в кэше или не изображение, то пропускаем ход
				continue
			}
		} else { // если Imgthumb creation
			if checkIfImgThumbInCache(v) || !checkIfImg(v) { // если файл существует в кэше или не изображение, то пропускаем ход
				continue
			}
		}
		if !checkIfImageExist(v) { // если исходный файл не существует на диск, то возврат с ошибкой
			glg.Errorf("[%s] %s not exist", glgFunc, v)
			return false
		}
		if thumbType == 0 {
			createThumb(v)
		} else {
			createImgThumb(v)
		}

		endTime := time.Now().Round(time.Second) // time check performance
		glg.Infof("[%s] %s %s", glgFunc, v, endTime.Sub(startTime))

	}
	return true
}

func thumbGoOverPriority() {
	for {
		if len(list2CreateThumbPriority) > 0 {
			glg.Infof("[thumbGoOverPriority] %d files to operate", len(list2CreateThumbPriority))
			for k, v := range list2CreateThumbPriority {
				//		fmt.Println("[queue, pri] ", v)
				if checkIfThumbInCache(v) || !checkIfImageExist(v) || !checkIfImg(v) { // если файл существует в кэше или исходник отсутствует на диске или это не изображение, то пропускаем ход
					list2CreateThumbPriority, _ = removeSlice(list2CreateThumbPriority, k)
					continue
				}
				startTime := time.Now().Round(time.Second) // time check performance
				createThumb(v)
				endTime := time.Now().Round(time.Second) // time check performance
				glg.Infof("[thumb, pri] %s %s", v, endTime.Sub(startTime))

				list2CreateThumbPriority, _ = removeSlice(list2CreateThumbPriority, k)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// удаление элемента из слайса
func removeSlice(s []string, index int) ([]string, error) {
	if index >= len(s) {
		return nil, errors.New("Out of Range Error")
	}
	//	return append(s[:index], s[index+1:]...), nil
	s[index] = s[len(s)-1]
	s = s[:len(s)-1]
	return s, nil
}

// EXIF check orientation
func checkJpegOrientation(path string) int {
	var imgOrient *tiff.Tag // exif tag для данные о повороте
	if checkIfJpeg(path) {
		f, err := os.Open(path) // загрузка файла
		check(err)
		defer f.Close()

		x, err := exif.Decode(f) // декодирование exif
		if err != nil {          // если отсутствует заголовок exif
			return 1
		} else { // если заголовок присутствует
			imgOrient, err = x.Get(exif.Orientation)
			if err != nil {
				return 1
			}
			return str2int(imgOrient.String())
		}
	} else {
		return 1
	}
}

// нормализует повернутое изображение
func normalizeOrientation(src image.Image, path string) image.Image {
	switch checkJpegOrientation(path) {
	case 6:
		src = imaging.Rotate270(src)
	case 8:
		src = imaging.Rotate90(src)
	case 3:
		src = imaging.Rotate180(src)
	}
	return src
}

// создание thumb
func createThumb(filename string) {
	src, err := imaging.Open(config.PathImages + filename)
	if err != nil {
		glg.Errorf("[createThumb] %s, open failed: %v", config.PathImages+filename, err)
		return
	}

	src = normalizeOrientation(src, config.PathImages+filename) // нормализация повернутого изображения
	//	! src = imaging.Fill(src, 155, 103, imaging.Center, imaging.Lanczos)
	src = imaging.Thumbnail(src, 155, 103, imaging.Lanczos)

	// Save the resulting image using JPEG format.
	saveFilename := config.PathCacheThumb + filename
	if !checkIfThumbInCache(saveFilename) { // если отсутствует папка, то создает
		os.Mkdir(config.PathCacheThumb+filepath.Dir(filename), 0644)
	}

	err = imaging.Save(src, saveFilename)
	if err != nil {
		glg.Errorf("[createThumb] %s,save failed: %v", config.PathImages+filename, err)
	}
}

// создание ImgThumb
func createImgThumb(filename string) {
	src, err := imaging.Open(config.PathImages + filename)
	if err != nil {
		glg.Errorf("[createImgThumb] %s, open failed: %v", config.PathImages+filename, err)
		return
	}

	src = normalizeOrientation(src, config.PathImages+filename) // нормализация повернутого изображения

	src = imaging.Fit(src, 800, 600, imaging.Box)
	//src = imaging.Fill(src, 800, 480, imaging.Center, imaging.Lanczos)

	// Save the resulting image using JPEG format.
	saveFilename := config.PathCacheImages + filename
	if !checkIfThumbInCache(saveFilename) { // если отсутствует папка, то создает
		os.Mkdir(config.PathCacheImages+filepath.Dir(filename), 0644)
	}

	err = imaging.Save(src, saveFilename)
	if err != nil {
		glg.Errorf("[createImgThumb] %s,save failed: %v", config.PathImages+filename, err)
	}
}

// преобразование строки в число
func str2int(val string) int {
	i, _ := strconv.Atoi(val)
	return i
}
