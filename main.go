package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/tkanos/gonfig"
)

type Configuration struct {
	Cookie               string
	CrawlIntervalSecond  int
	DailyGoal            int
	EasyScore            int
	MediumScore          int
	HardScore            int
	StartTimeOfTheNewDay int
}

type Problem struct {
	ID        int64
	Title     string
	Level     int64
	TimeStamp int64
	RunTime   string
	Memory    string
	State     string
}

var subl *widgets.List
var acl *widgets.List
var curList *widgets.List //current selected list
var msg *widgets.Paragraph
var clock *widgets.Paragraph
var todaybc *widgets.BarChart
var weekSubsbc *widgets.StackedBarChart
var configuration Configuration
var progress *widgets.Gauge
var weekACsbc *widgets.StackedBarChart
var problemMap = make(map[string]Problem)
var curSubmissionData = make([]Problem, 0)
var msgDefault = "Press 'q' to quit, press 'r' to refresh, KeyArrow to view Sub/AC list"

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: lcdash ./lcdashconfig.json  \nYou can download the config file template in https://github.com/yangchenxi/LeetcodeDashboard")
		return
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}

	defer ui.Close()
	subl = widgets.NewList()
	curList = subl
	msg = widgets.NewParagraph()
	clock = widgets.NewParagraph()
	acl = widgets.NewList()
	todaybc = widgets.NewBarChart()
	progress = widgets.NewGauge()
	weekACsbc = widgets.NewStackedBarChart()
	weekSubsbc = widgets.NewStackedBarChart()
	InitUI()
	msg.Text = "Loading configuration..."
	ui.Render(msg)
	err := gonfig.GetConf(os.Args[1], &configuration)

	if err != nil {
		panic(err)
	}
	//
	//operations

	GetAllProblems()
	RefreshData()
	//

	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	refreshTicker := time.NewTicker(time.Duration(configuration.CrawlIntervalSecond) * time.Second).C

	for {
		select {
		case e := <-uiEvents:
			switch e.ID { // event string/identifier
			case "q", "<C-c>": // press 'q' or 'C-c' to quit
				return
			case "r": //press 'r' to refresh data
				RefreshData()
				RefreshGrid()

			case "<Resize>":
				RefreshGrid()
			case "<Up>":
				curList.ScrollUp()
				ui.Render(curList)
			case "<Down>":
				curList.ScrollDown()
				ui.Render(curList)
			case "<Right>":

				if curList == subl {
					curList = acl
					subl.BorderStyle = ui.NewStyle(ui.ColorWhite)
					curList.BorderStyle = ui.NewStyle(ui.ColorGreen)
					ui.Render(curList, subl)
				}
			case "<Left>":
				if curList == acl {
					curList = subl
					acl.BorderStyle = ui.NewStyle(ui.ColorWhite)
					curList.BorderStyle = ui.NewStyle(ui.ColorGreen)
					ui.Render(curList, acl)
				}
			}
		// use Go's built-in tickers for updating and drawing data
		case <-ticker:
			//update clock
			UpdateClock()
		case <-refreshTicker:
			RefreshData()
		}

	}
}

func UpdateClock() {
	clock.Text = time.Now().Format(time.RFC850)

	ui.Render(subl, clock)
}

func GetSubmissions() error {
	msg.Text = "Requesting Submission Data..."
	ui.Render(msg)
	curSubmissionData = make([]Problem, 0)
	hasNext := true
	lastKey := ""
	CurTime := time.Now()
	LastTimeStamp := CurTime.Unix()
	TargetTimeStamp := CurTime.Unix() - int64(CurTime.Hour()-configuration.StartTimeOfTheNewDay)*3600 - 86400*8 //nowTs-(nowHour-startHour)*tshour-7daysTs

	for hasNext && LastTimeStamp > TargetTimeStamp {
		requestURL := "https://leetcode.com/api/submissions/?offset=0&limit=20&lastkey=" + lastKey
		//println(requestURL)
		client := &http.Client{}

		req, err := http.NewRequest("GET", requestURL, nil)
		req.Header.Set("cookie", configuration.Cookie)
		if err != nil {
			msg.Text = "Error:" + err.Error()
			ui.Render(msg)
			log.Fatalln(err)
			return err
		}

		resp, err := client.Do(req)

		if err != nil {
			msg.Text = "Error:" + err.Error()
			ui.Render(msg)
			return err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			msg.Text = "Error:" + err.Error()
			ui.Render(msg)
			return err
		}

		if resp.StatusCode != 200 {
			msg.Text = "Error:" + "Http Respcode Error, Code " + strconv.Itoa(resp.StatusCode)
			ui.Render(msg)
			return errors.New("Http Respcode Error, Code" + strconv.Itoa(resp.StatusCode))
		}

		hasNext, _ = jsonparser.GetBoolean(body, "has_next")
		lastKey, _ = jsonparser.GetString(body, "last_key")
		_, err = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

			ProblemTS, err := jsonparser.GetInt(value, "timestamp")
			if err != nil {
				msg.Text = "Error:" + err.Error()
				ui.Render(msg)
				return

			}
			ProblemStat, err := jsonparser.GetString(value, "status_display")
			if err != nil {
				msg.Text = "Error:" + err.Error()
				ui.Render(msg)
				return

			}
			ProblemRT, err := jsonparser.GetString(value, "runtime")
			if err != nil {
				msg.Text = "Error:" + err.Error()
				ui.Render(msg)
				return

			}
			ProblemTitle, err := jsonparser.GetString(value, "title")
			if err != nil {
				msg.Text = "Error:" + err.Error()
				ui.Render(msg)
				return
			}
			ProblemMem, err := jsonparser.GetString(value, "memory")
			if err != nil {
				msg.Text = "Error:" + err.Error()
				ui.Render(msg)
				return

			}
			prob := problemMap[ProblemTitle]
			prob.Memory = ProblemMem
			prob.RunTime = ProblemRT
			prob.TimeStamp = ProblemTS
			prob.State = ProblemStat
			LastTimeStamp = ProblemTS
			curSubmissionData = append(curSubmissionData, prob)

		}, "submissions_dump")
		if err != nil {
			msg.Text = "Error:" + err.Error()
			ui.Render(msg)
			return err
		}

	}

	msg.Text = msgDefault
	ui.Render(msg)
	return nil
}

func RefreshData() {
	err := GetSubmissions()
	if err != nil {
		return
	}
	//get Today Submissions:
	msg.Text = "Processing Data..."
	ui.Render(msg)
	todaySub := make([]Problem, 0)
	twoBSub := make([]Problem, 0)
	threeBSub := make([]Problem, 0)
	fourBSub := make([]Problem, 0)
	fiveBSub := make([]Problem, 0)
	sixBSub := make([]Problem, 0)
	sevenBSub := make([]Problem, 0)
	CurTime := time.Now()

	var TodayTargetTimeStamp int64
	if configuration.StartTimeOfTheNewDay < CurTime.Hour() {
		//set today's start time of day

		TodayTargetTimeStamp = CurTime.Add(-time.Hour/2).Round(time.Hour).Unix() - int64(CurTime.Hour()-configuration.StartTimeOfTheNewDay)*3600
	} else {
		//set yesterday's start time of day
		TodayTargetTimeStamp = CurTime.Add(-time.Hour/2).Round(time.Hour).Add(-time.Hour*24).Unix() - int64(CurTime.Hour()-configuration.StartTimeOfTheNewDay)*3600

	}
	for _, data := range curSubmissionData {
		if data.TimeStamp >= TodayTargetTimeStamp {
			todaySub = append(todaySub, data)
		} else if data.TimeStamp < TodayTargetTimeStamp && data.TimeStamp >= TodayTargetTimeStamp-86400 {
			//2 day ago
			twoBSub = append(twoBSub, data)
		} else if data.TimeStamp < TodayTargetTimeStamp-86400 && data.TimeStamp >= TodayTargetTimeStamp-86400*2 {
			//3 day ago
			threeBSub = append(threeBSub, data)
		} else if data.TimeStamp < TodayTargetTimeStamp-86400*2 && data.TimeStamp >= TodayTargetTimeStamp-86400*3 {
			//4 day ago
			fourBSub = append(fourBSub, data)
		} else if data.TimeStamp < TodayTargetTimeStamp-86400*3 && data.TimeStamp >= TodayTargetTimeStamp-86400*4 {
			//5 day ago
			fiveBSub = append(fiveBSub, data)
		} else if data.TimeStamp < TodayTargetTimeStamp-86400*4 && data.TimeStamp >= TodayTargetTimeStamp-86400*5 {
			//6 day ago
			sixBSub = append(sixBSub, data)
		} else if data.TimeStamp < TodayTargetTimeStamp-86400*5 && data.TimeStamp >= TodayTargetTimeStamp-86400*6 {
			//7 day ago
			sevenBSub = append(sevenBSub, data)
		}
	}

	//submission List
	uiSubtodayData := make([]string, len(todaySub))

	for i, subTodayData := range todaySub {

		uiSubtodayData[i] = "[" + strconv.FormatInt(subTodayData.ID, 10) + "] [" + subTodayData.State + "-" + subTodayData.Title + "]"
		if subTodayData.State == "Accepted" {
			uiSubtodayData[i] = uiSubtodayData[i] + "(fg:green)"
		} else {
			uiSubtodayData[i] = uiSubtodayData[i] + "(fg:red)"
		}
	}
	subl.Title = "Submissions(" + strconv.Itoa(len(todaySub)) + ")"
	subl.Rows = uiSubtodayData
	ui.Render(subl)
	//AC List
	easyNum := make(map[int64]bool)
	mediumNum := make(map[int64]bool)
	hardNum := make(map[int64]bool)
	//===

	uiActodayData := make([]string, 0)
	for _, acTodayData := range todaySub {
		if acTodayData.State == "Accepted" {
			var tmp string
			tmp = "[" + strconv.FormatInt(acTodayData.ID, 10) + "] [" + acTodayData.RunTime + "-" + acTodayData.Memory + "-" + acTodayData.Title + "]"
			if acTodayData.Level == 1 {
				tmp += "(fg:green)"
				easyNum[acTodayData.ID] = true
			} else if acTodayData.Level == 2 {
				tmp += "(fg:yellow)"
				mediumNum[acTodayData.ID] = true
			} else {
				tmp += "(fg:red)"
				hardNum[acTodayData.ID] = true
			}
			uiActodayData = append(uiActodayData, tmp)
		}
	}
	acl.Title = "Accepted(" + strconv.Itoa(len(uiActodayData)) + ")"
	acl.Rows = uiActodayData
	ui.Render(acl)
	//Solved Today
	easy := len(easyNum)
	medium := len(mediumNum)
	hard := len(hardNum)

	todaybc.Data = []float64{float64(easy), float64(medium), float64(hard)}
	ui.Render(todaybc)
	//=======progress bar
	percent := (easy*configuration.EasyScore + medium*configuration.MediumScore + hard*configuration.HardScore) * 100 / configuration.DailyGoal
	if percent > 100 {
		percent = 100
	}
	progress.Percent = percent
	ui.Render(progress)
	//===data processing

	//=======weekly sub
	weeklabel := []string{CurTime.Add(-24 * time.Hour * 6).Weekday().String(), CurTime.Add(-24 * time.Hour * 5).Weekday().String(), CurTime.Add(-24 * time.Hour * 4).Weekday().String(), CurTime.Add(-24 * time.Hour * 3).Weekday().String(), CurTime.Add(-24 * time.Hour * 2).Weekday().String(), CurTime.Add(-24 * time.Hour * 1).Weekday().String(), CurTime.Weekday().String()}
	weekSubsbc.Labels = weeklabel
	weekACsbc.Labels = weeklabel
	weekSubsbc.Data = make([][]float64, 7)
	weekACsbc.Data = make([][]float64, 7)
	total, ea, me, ha := getSolvedData(todaySub)
	weekSubsbc.Data[6] = []float64{float64(total), float64(len(todaySub) - total)}
	weekACsbc.Data[6] = []float64{float64(ea), float64(me), float64(ha)}
	total, ea, me, ha = getSolvedData(twoBSub)
	weekSubsbc.Data[5] = []float64{float64(total), float64(len(twoBSub) - total)}
	weekACsbc.Data[5] = []float64{float64(ea), float64(me), float64(ha)}
	total, ea, me, ha = getSolvedData(threeBSub)
	weekSubsbc.Data[4] = []float64{float64(total), float64(len(threeBSub) - total)}
	weekACsbc.Data[4] = []float64{float64(ea), float64(me), float64(ha)}
	total, ea, me, ha = getSolvedData(fourBSub)
	weekSubsbc.Data[3] = []float64{float64(total), float64(len(fourBSub) - total)}
	weekACsbc.Data[3] = []float64{float64(ea), float64(me), float64(ha)}
	total, ea, me, ha = getSolvedData(fiveBSub)
	weekSubsbc.Data[2] = []float64{float64(total), float64(len(fiveBSub) - total)}
	weekACsbc.Data[2] = []float64{float64(ea), float64(me), float64(ha)}
	total, ea, me, ha = getSolvedData(sixBSub)
	weekSubsbc.Data[1] = []float64{float64(total), float64(len(sixBSub) - total)}
	weekACsbc.Data[1] = []float64{float64(ea), float64(me), float64(ha)}
	total, ea, me, ha = getSolvedData(sevenBSub)
	weekSubsbc.Data[0] = []float64{float64(total), float64(len(sevenBSub) - total)}
	weekACsbc.Data[0] = []float64{float64(ea), float64(me), float64(ha)}
	//=========
	ui.Render(weekSubsbc, weekACsbc)

	msg.Text = msgDefault
	ui.Render(msg)
}

//Return TotalAC, ACEasy ACmedium ACHard
func getSolvedData(todaySub []Problem) (int, int, int, int) {
	//AC List
	total := 0
	easy := 0
	medium := 0
	hard := 0
	//===

	for _, acTodayData := range todaySub {
		if acTodayData.State == "Accepted" {
			total++
			if acTodayData.Level == 1 {
				easy++
			} else if acTodayData.Level == 2 {
				medium++
			} else {
				hard++
			}

		}
	}
	return total, easy, medium, hard
}

func GetAllProblems() {
	msg.Text = "Loading Problems...."
	ui.Render(msg)
	requestURL := "https://leetcode.com/api/problems/all/"
	//println(requestURL)
	client := &http.Client{}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		msg.Text = "Error:" + err.Error()
		ui.Render(msg)
		log.Fatalln(err)
		return
	}

	resp, err := client.Do(req)

	if err != nil {
		msg.Text = "Error:" + err.Error()
		ui.Render(msg)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg.Text = "Error:" + err.Error()
		ui.Render(msg)
		return
	}

	if resp.StatusCode != 200 {
		msg.Text = "Error:" + err.Error()
		ui.Render(msg)
		return
	}
	_, err = jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {

		ProblemID, err := jsonparser.GetInt(value, "stat", "question_id")
		if err != nil {
			msg.Text = "Error:" + err.Error()
			ui.Render(msg)
			return

		}
		DiffLevel, err := jsonparser.GetInt(value, "difficulty", "level")
		if err != nil {
			msg.Text = "Error:" + err.Error()
			ui.Render(msg)
			return

		}
		ProblemTitle, err := jsonparser.GetString(value, "stat", "question__title")
		if err != nil {
			msg.Text = "Error:" + err.Error()
			ui.Render(msg)
			return

		}
		problemMap[ProblemTitle] = Problem{
			Title: ProblemTitle,
			ID:    ProblemID,
			Level: DiffLevel,
		}

	}, "stat_status_pairs")
	if err != nil {
		msg.Text = "Error:" + err.Error()
		ui.Render(msg)
		return
	}

	msg.Text = msgDefault
	ui.Render(msg)
	return

}

func RefreshGrid() {
	//======
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(1.0/10,
			ui.NewCol(3.0/5, msg),
			ui.NewCol(2.0/5, clock),
		),
		ui.NewRow(2.0/5,
			ui.NewCol(2.0/5, subl),
			ui.NewCol(2.0/5, acl),
			ui.NewCol(1.0/5, todaybc),
		),
		ui.NewRow(2.0/5,
			ui.NewCol(1.0/2, weekSubsbc),
			ui.NewCol(1.0/2, weekACsbc),
		),
		ui.NewRow(1.0/10, progress),
	)
	ui.Render(grid)
}

func InitUI() {

	//Message

	msg.Title = "Message"
	msg.Text = msgDefault
	//Clock
	clock.Title = "Current Time"

	subl.Title = "Submissions(0)"

	acl.Title = "Accepted(0)"

	//Today Solved Unique Questions:
	todaybc.Title = "Solved Today"
	todaybc.Labels = []string{"Easy", "Medium", "Hard"}
	todaybc.BarWidth = 6
	todaybc.BarGap = 2
	todaybc.LabelStyles[0] = ui.NewStyle(ui.ColorGreen)
	todaybc.BarColors[0] = ui.ColorGreen
	todaybc.NumStyles[0] = ui.NewStyle(ui.ColorRed)
	todaybc.BarColors[1] = ui.ColorYellow
	todaybc.LabelStyles[1] = ui.NewStyle(ui.ColorYellow)
	todaybc.NumStyles[1] = ui.NewStyle(ui.ColorRed)
	todaybc.BarColors[2] = ui.ColorRed
	todaybc.LabelStyles[2] = ui.NewStyle(ui.ColorRed)
	todaybc.NumStyles[2] = ui.NewStyle(ui.ColorRed)

	//week submission stacked barchart stack:(AC/TLE/wrong Answer/Runtime Error/Compile Error/MemoryLimitExceded)

	weekSubsbc.Title = "Submissions in last 7 days (AC/Error)"
	weekSubsbc.BarColors = []ui.Color{ui.ColorGreen, ui.ColorRed}
	weekSubsbc.NumStyles = []ui.Style{ui.NewStyle(ui.ColorBlue)}
	weekSubsbc.BarWidth = 7
	weekSubsbc.BarGap = 2
	//week AC stacked barchart stack:(easy/medium/hard)

	weekACsbc.Title = "Accepted in last 7 days (Easy Medium Hard)"

	weekACsbc.BarWidth = 7
	weekACsbc.BarGap = 2

	//Today target progress

	progress.Title = "Today's Progress"
	progress.Percent = 0
	progress.BarColor = ui.ColorGreen
	progress.BorderStyle.Fg = ui.ColorWhite
	progress.TitleStyle.Fg = ui.ColorCyan

	RefreshGrid()

}
