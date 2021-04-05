package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/kpango/glg"
)

type FolderStruct struct {
	Name          string
	IsDir         int
	NeedToRefresh bool
}

type NavStruct struct {
	Name    string
	Href    string
	Current bool
}

type PagesStruct struct {
	Name    string
	Current bool
}

type IndexData struct {
	NavData    []NavStruct
	FolderData []FolderStruct
	PagesData  []PagesStruct
	ArrowPrev  int
	ArrowNext  int
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//	glg.Infof("[URL] %s", r.URL.Path) // путь обращения к вебсерверу

	if r.URL.Path == "/" {
		webShowFolder(w, r, config.PathImages)
		glg.Infof("[URL] %s", r.URL.Path) // путь обращения к вебсерверу
		return
	} else if checkIfInData, _ := AssetInfo(config.PathWWW + r.URL.Path); checkIfInData != nil { // если www файл существует
		data, _ := Asset(config.PathWWW + r.URL.Path)
		w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(r.URL.Path)))
		fmt.Fprintf(w, "%s", data)
		return
	} else if r.URL.Path == config.DataIconFolderName { // иконка папки
		fmt.Fprintf(w, "%s", config.DataIconFolder) // выводим содержимое файла
		return
	} else if r.URL.Path == config.DataIconUnknownName { // иконка необрабатываемого файла
		fmt.Fprintf(w, "%s", config.DataIconUnknown) // выводим содержимое файла
		return
	} else if checkIfImg(r.URL.Path) { // если это изображение
		if checkIfImageExist(r.URL.Path[1:]) { // если оно существует
			if isReqThumb(r) { // если запрашивается thumb
				if checkIfThumbInCache(r.URL.Path[1:]) { // если существует файл в кэше
					fmt.Fprintf(w, "%s", readFile(config.PathCacheThumb+r.URL.Path[1:])) // выводим содержимое файла
				} else {
					if len(list2CreateThumbPriority) < 30000 { // защита от переполнения памяти
						list2CreateThumbPriority = append(list2CreateThumbPriority, r.URL.Path[1:])
					}
					fmt.Fprintf(w, "%s", config.DataIconLoading) // выводим содержимое файла заглушки
				}
			} else if checkIfImgThumbInCache(r.URL.Path[1:]) { // если запрашивается изображение и он есть в кэше
				fmt.Fprintf(w, "%s", readFile(config.PathCacheImages+r.URL.Path[1:])) // выводим содержимое файла
			} else { // если запрашивается просто изображение
				fmt.Fprintf(w, "%s", readFile(config.PathImages+r.URL.Path[1:])) // выводим содержимое файла
			}
		} else { // вывод содержимого папки
			webShowFolder(w, r, config.PathImages)
		}
		return
	} else {
		webShowFolder(w, r, config.PathImages)
		glg.Infof("[URL] %s", r.URL.Path) // путь обращения к вебсерверу
		return
	}
	http.NotFound(w, r)
	return
}

func webShowFolder(w http.ResponseWriter, r *http.Request, baseDir string) {
	dataTemplateMain, _ := Asset(config.PathWWW + "/index.html") // загружаем основной шаблон

	funcMapEach := template.FuncMap{
		"each": func(interval, n int) bool {
			return (n+1)%interval == 0
		},
	}
	tMain, _ := template.New("main").Funcs(funcMapEach).Parse(fmt.Sprintf("%s", dataTemplateMain))

	url := webCleanPathInUrl(r)                // очистка пути из url'а
	pathLevel := strings.Count(url, "/")       // подсчет текущего уровня
	folder, _ := ioutil.ReadDir(baseDir + url) // чтение содержимого папки с фотографиями

	navData := webShowNavData(url, pathLevel)                                                                                                              // формируем данные для навигации
	folderData := webShowFolderData(pathLevel, folder, r)                                                                                                  // формируем данные для содержимого папки
	webShowFolderPageStart, webShowFolderPageFinish, webShowFolderPagePages, webShowFolderPageCurrent := webShowFolderPreparePages(r, 36, len(folderData)) // подготовка блока данных постраничного вывода
	pagesData := webShowFolderPreparePagesSlice(webShowFolderPageCurrent, webShowFolderPagePages)                                                          // подготовка slice'а для наполнения номерами страниц
	ArrowPrevInt, ArrowNextInt := webShowFolderPrepareArrows(webShowFolderPageCurrent, webShowFolderPagePages)                                             // подготовка данных для стрелок листания страниц

	// обрезка выводимых данных до данных для текущей страницы
	folderData = folderData[webShowFolderPageStart:webShowFolderPageFinish]

	// вывод по шаблону
	tMain.Execute(w, IndexData{NavData: navData, FolderData: folderData, PagesData: pagesData, ArrowPrev: ArrowPrevInt, ArrowNext: ArrowNextInt})
}

// подготовка списка для навигации
func webShowNavData(url string, pathLevel int) []NavStruct {
	navData := make([]NavStruct, 0) // подготовка slice'а для наполнения поля навигации
	var navDataFull string
	var navDataCurrent bool
	navUrl := strings.Split(url, "/")
	for i := 0; i < pathLevel; i++ {
		navDataFull = navDataFull + navUrl[i] + "/"
		if i == pathLevel-1 {
			navDataCurrent = true
		} else {
			navDataCurrent = false
		}
		navData = append(navData, NavStruct{Name: navUrl[i], Href: navDataFull, Current: navDataCurrent})
	}
	return navData
}

// подготовка списка содержимого папки
func webShowFolderData(pathLevel int, folder []os.FileInfo, r *http.Request) []FolderStruct {
	folderData := make([]FolderStruct, 0) // подготовка slice'а для наполнения содержимым папки
	if pathLevel > 0 {                    // если корень папки
		folderData = append(folderData, FolderStruct{Name: "..", IsDir: 0, NeedToRefresh: false})
	}

	for _, v := range folder { // добавляем сначала только папки
		if v.IsDir() && v.Name()[0:1] != "." && v.Name()[0:1] != "@" {
			folderData = append(folderData, FolderStruct{Name: v.Name(), IsDir: 1, NeedToRefresh: false})
		}
	}
	for _, v := range folder { // добавляем уже файлы
		if !v.IsDir() && v.Name()[0:1] != "." && v.Name()[0:1] != "@" {
			if checkIfImg(r.URL.Path[:1] + v.Name()) { // если изображение
				folderData = append(folderData, FolderStruct{Name: v.Name(), IsDir: 2, NeedToRefresh: !checkIfThumbInCache(r.URL.Path[:1] + v.Name())})
				//			} else { // если другой файл
				//				folderData = append(folderData, FolderStruct{Name: v.Name(), IsDir: 3, NeedToRefresh: false})
			}
		}
	}
	return folderData
}

// подготовка slice'а для номеров страниц
func webShowFolderPreparePagesSlice(webShowFolderPageCurrent int, webShowFolderPagePages int) []PagesStruct {
	pagesData := make([]PagesStruct, 0) // подготовка slice'а для наполнения номерами страниц
	if webShowFolderPagePages > 1 {
		for j := 0; j < webShowFolderPagePages; j++ { // создаем срез с номерами страниц и указанием текущей
			if j == (webShowFolderPageCurrent - 1) {
				pagesData = append(pagesData, PagesStruct{Name: strconv.Itoa(j + 1), Current: true}) // текущая страница
			} else {
				pagesData = append(pagesData, PagesStruct{Name: strconv.Itoa(j + 1), Current: false}) // все остальные страницы
			}
		}
	}
	return pagesData
}

// возвращает номера предыдущей и следующей страниц
func webShowFolderPrepareArrows(webShowFolderPageCurrent int, webShowFolderPagePages int) (int, int) {
	var ArrowPrevInt, ArrowNextInt int
	if webShowFolderPageCurrent != 1 {
		ArrowPrevInt = webShowFolderPageCurrent - 1
	} else {
		ArrowPrevInt = 0
	}
	if webShowFolderPageCurrent < webShowFolderPagePages {
		ArrowNextInt = webShowFolderPageCurrent + 1
	} else {
		ArrowNextInt = 0
	}
	return ArrowPrevInt, ArrowNextInt
}

// подсчитываем начало, конец постраничного среза
// на выходе: startPos, FinishPos, всего страниц (1+), текущая страница (1+)
func webShowFolderPreparePages(r *http.Request, limit int, folderDataLen int) (int, int, int, int) {
	var webShowFolderPage, webShowFolderPageStart, webShowFolderPagePages, webShowFolderPageFinish int
	webShowFolderPagePages = calcPagesNumber(folderDataLen, limit)

	if _, ok := r.URL.Query()["p"]; ok { // если в URL передана переменная p
		webShowFolderPage = int(math.Abs(float64(str2int(strings.Join(r.URL.Query()["p"], ""))))) // берем целое число по модулю
	} else { // если нет переменной, то по умолчанию ставим первую страницы
		webShowFolderPage = 1
	}

	if webShowFolderPage > webShowFolderPagePages { // если переданный в url номер страницы больше общего количества страниц
		webShowFolderPage = 1
	}

	if (webShowFolderPage-1)*36 > folderDataLen || webShowFolderPage == 0 { // если переданная страницы превышает общее количество записей, то стартовая позиция равна первой странице
		webShowFolderPageStart = 0
	} else { // иначе расчитываем
		webShowFolderPageStart = (webShowFolderPage - 1) * 36
	}

	if webShowFolderPageStart+36 > folderDataLen { // если количество записей от начала старта превышает общее количество среза, то ограничиваемся максимальным размером среза
		webShowFolderPageFinish = folderDataLen
	} else { // иначе расчитываем
		webShowFolderPageFinish = webShowFolderPageStart + 36
	}
	return webShowFolderPageStart, webShowFolderPageFinish, webShowFolderPagePages, webShowFolderPage
}

// очищаем путь в URL
func webCleanPathInUrl(r *http.Request) string {
	dir, _ := filepath.Split(r.URL.Path[1:])
	dir = path.Clean(dir)
	if dir[0:1] != "." { // еслли не корень папки, то добавляем слэш в конце
		dir = dir + "/"
	}
	return dir
}

// чтение содержимого файла
func readFile(filename string) string {
	configFileText, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	str := string(configFileText)
	return str
}

// проверка является ли запрашиваемый файл изображением
func checkIfImg(path string) bool {
	ext := filepath.Ext(path)
	imagesExt := make(map[string]bool)
	imagesExt[".jpg"] = true
	imagesExt[".jpeg"] = true
	imagesExt[".gif"] = true
	imagesExt[".png"] = true

	if _, ok := imagesExt[strings.ToLower(ext)]; ok {
		return true
	}
	return false
}

// если jpeg
func checkIfJpeg(path string) bool {
	ext := filepath.Ext(path)
	imagesExt := make(map[string]bool)
	imagesExt[".jpg"] = true
	imagesExt[".jpeg"] = true

	if _, ok := imagesExt[strings.ToLower(ext)]; ok {
		return true
	}
	return false
}

// проверка существует ли изображение в кэше
func checkIfThumbInCache(path string) bool {
	if _, err := os.Stat(config.PathCacheThumb + path); err == nil {
		return true
	}
	return false
}

// проверка существует ли изображение в кэше
func checkIfImgThumbInCache(path string) bool {
	if _, err := os.Stat(config.PathCacheImages + path); err == nil {
		return true
	}
	return false
}

// проверка существует ли изображение на диске
func checkIfImageExist(path string) bool {
	if _, err := os.Stat(config.PathImages + path); err == nil {
		return true
	}
	return false
}

// проверка запрашивается ли thumbnail
func isReqThumb(r *http.Request) bool {
	if _, ok := r.URL.Query()["thumbnail"]; ok {
		return true
	}
	return false
}

// возвращает количество страниц
func calcPagesNumber(summ int, perPage int) int {
	var pages int
	pages = summ / perPage
	if summ%perPage > 0 {
		pages = pages + 1
	}
	return pages
}
