// photo_cache_service
// TODO:
// - как-то сделать обновление в html картин после создания thumb к ним, видимо заменять весь блок div class col-

package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/hjson/hjson-go"
	"github.com/kpango/glg"
)

type ConfigStruct struct {
	PathImages      string
	PathImagesLen   int
	PathCacheThumb  string
	PathCacheImages string
	PathWWW         string

	DataIconLoading     []byte
	DataIconLoadingName string
	DataIconFolder      []byte
	DataIconFolderName  string
	DataIconUnknown     []byte
	DataIconUnknownName string

	Seconds2RefreshCache   int
	Seconds2WaitAfterError int
}

var config ConfigStruct

var list2CreateThumb []string         // слайс для списка файлов на создание thumb
var list2CreateThumbPriority []string // слайс для приоритетного списка файлов на создание thumb

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	setConfig() // конфигурация

	//	установки log-файла
	infolog := glg.FileWriter("photo_cache_service.log", 0666) // открытие лог.файла
	defer infolog.Close()
	customErrLevel := "FINE"
	//	customErrLevel := "CRIT"
	//	customErrLevel := "DEBUG"
	glg.Get().
		//SetMode(glg.BOTH). // default is STD
		AddWriter(infolog).
		SetWriter(infolog).
		SetLevelColor(customErrLevel, glg.Red) // set color output to user custom level
	//	завершение установки log-файла

	glg.Infof("%s", "started")

	go updateCache()
	go thumbGoOverPriority()

	http.HandleFunc("/", ServeHTTP)
	err := http.ListenAndServe(":9090", nil)
	check(err)
}

func updateCache() {
	for {
		list2CreateThumb = list2CreateThumb[:0] // очистка списка на создания thumb
		glg.Infof("%s", "[listdir] started")
		listDir(config.PathImages, config.PathCacheThumb) // создание списка на создание thumb

		glg.Info("[thumb] start cache update")
		if thumbGoOver(0) { // запуск создания thumb
			glg.Infof("[thumb] cache updated")

			list2CreateThumb = list2CreateThumb[:0] // очистка списка на создания thumb
			glg.Infof("%s", "[listdirImg] started")
			listDir(config.PathImages, config.PathCacheImages) // создание списка на создание thumb
			glg.Info("[thumbImg] start cache update")
			if thumbGoOver(1) { // обновление кэша изображений
				glg.Infof("[thumbImg] image cache updated, wait %d second before rescan again", config.Seconds2RefreshCache)
				time.Sleep(time.Duration(config.Seconds2RefreshCache) * time.Second) // timeout, если все нормально
			} else {
				glg.Infof("[thumbImg] error, wait %d second before update cache again", config.Seconds2WaitAfterError)
				time.Sleep(time.Duration(config.Seconds2WaitAfterError) * time.Second)
			}
		} else {
			glg.Infof("[thumb] error, wait %d second before update cache again", config.Seconds2WaitAfterError)
			time.Sleep(time.Duration(config.Seconds2WaitAfterError) * time.Second)
		}
	}
}

// выводит содержимое папки, рекурсивная
func listDir(baseDir string, cachePath string) {
	baseDirClean := baseDir[config.PathImagesLen:]              // делаем относительный путь к фотографиям
	foldersCache, _ := ioutil.ReadDir(cachePath + baseDirClean) // чтение содержимого папки с кэшем
	folders, _ := ioutil.ReadDir(baseDir)                       // чтение содержимого папки с фотографиями
	refreshCache(folders, foldersCache, baseDirClean)           // обновление кеша, удаление лишних файлов и папок
	for _, f := range folders {                                 // перебор значений папки
		if f.Name()[0:1] != "." && f.Name()[0:1] != "@" { // пропускаем скрыте папки с . в начале имени
			if f.IsDir() == true {
				if !checkIfExist(cachePath + baseDirClean + f.Name()) { // если папка отсутствует в кеше, то создается
					os.Mkdir(cachePath+baseDirClean+f.Name(), 0644)
				}
				listDir(baseDir+f.Name()+`/`, cachePath) // если значение это папка, то вход внутрь
			} else {
				if !checkIfExist(cachePath+baseDirClean+f.Name()) && checkIfImg(f.Name()) { // если файл отсутствует в кеше, то создается
					list2CreateThumb = append(list2CreateThumb, baseDirClean+f.Name())
				}
			}
		}
	}
}

// проверка существует ли файл/папка
func checkIfExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func checkIfExist4Log(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "- "
	}
	return "+ "
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// актуализация кэша в указанной указанной папке
func refreshCache(folders []os.FileInfo, foldersCache []os.FileInfo, baseDirClean string) {
	foldersH := make(map[string]string)
	foldersCacheH := make(map[string]string)

	// переводит содержимое папки в хэш
	for _, f := range folders {
		foldersH[f.Name()] = strconv.FormatBool(f.IsDir())
	}

	// переводит содержимое папки кеша в хэш
	for _, f := range foldersCache {
		foldersCacheH[f.Name()] = strconv.FormatBool(f.IsDir())
	}

	// сравнение папок с фотографиями и кэшом и выделение тех, которые в кеше лишние
	for k, v := range foldersCacheH {
		if foldersCacheH[k] != foldersH[k] { // если в кеше лишний файл/папка
			if v == "true" { // удалить, если папка
				os.RemoveAll(config.PathCacheThumb + baseDirClean + k)
			} else { // удалить, если файл
				os.Remove(config.PathCacheThumb + baseDirClean + k)
			}
		}
	}
}

func setConfig() {
	mainConfig := readCfgFile("config.json") // чтение конфигурации

	config.PathImages = mainConfig["PathImages"].(string) // только абсолютный путь со слэшем в конце
	config.PathImagesLen = len(config.PathImages)
	config.PathCacheThumb = mainConfig["PathCacheThumb"].(string)   // со слэшем в конце
	config.PathCacheImages = mainConfig["PathCacheImages"].(string) // со слэшем в конце
	config.PathWWW = mainConfig["PathWWW"].(string)                 // путь до html, без слеша в конце

	config.DataIconFolderName = mainConfig["DataIconFolderName"].(string)
	config.DataIconLoadingName = mainConfig["DataIconLoadingName"].(string)
	config.DataIconUnknownName = mainConfig["DataIconUnknownName"].(string)
	config.DataIconFolder, _ = Asset(config.PathWWW + config.DataIconFolderName)
	config.DataIconLoading, _ = Asset(config.PathWWW + config.DataIconLoadingName)
	config.DataIconUnknown, _ = Asset(config.PathWWW + config.DataIconUnknownName)

	config.Seconds2RefreshCache = int(mainConfig["Seconds2RefreshCache"].(float64))     // пауза между актуализацией кэша
	config.Seconds2WaitAfterError = int(mainConfig["Seconds2WaitAfterError"].(float64)) // пауза после ошибки для повторного обновления кэша
}

// чтение конфиг файла
func readCfgFile(filename string) map[string]interface{} {
	configFileText, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var dat map[string]interface{}
	// Decode and a check for errors.
	if err := hjson.Unmarshal(configFileText, &dat); err != nil {
		panic(err)
	}

	return dat
}
