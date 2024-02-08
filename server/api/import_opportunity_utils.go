package api

import (
	"fmt"
	"mime/multipart"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/focalboard/server/model"
	mmModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/xuri/excelize/v2"
)

type Opportunity struct {
	OpportunityId                string
	ReportingStatus              string
	FYQtr                        string
	CG                           string
	MasterClientName             string
	OpportunityName              string
	OpportunityDetailDescription string
	IndustrySegment              string
	Stage                        string
	TotalCurrentRevenue          string
	WinProbability               string
	SellingCountry               string
	Level2                       string
	SalesCapture                 string
	CreateDate                   string
	Revenue                      string
}

const (
	opportunityIdKey       = "az4n94b4qs14czh8oz65iy5oqch"
	stageKey               = "a5hwxjsmkn6bak6r7uea5bx1kwc"
	masterClientNameKey    = "ainpw47babwkpyj77ic4b9zq9xr"
	totalCurrentRevenueKey = "a4fbeujhoww8rhqh8jdw339nukh"
	winProbabilityKey      = "af7odq9ibb9f8rmap7wtdje5yua"
	salesCaptureKey        = "auoepp7uiphkaypcx3k34icuoxa"
	cgKey                  = "af1dyxxzjh8ycihtw6nx1tb7mty"
	level2Key              = "a3ait3cuckgx19pshzxyyx758fe"
	createDateKey          = "ay3wkiaubujrfe4t3cg8y4gr4kh"
)

func readOpportunitiesFromFile(file multipart.File, opportunities *[]Opportunity) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		fmt.Println("Fehler!!!")
		fmt.Println(err)
		return
	}
	defer file.Close()

	rows, err := f.GetRows("MMS Data")
	_ = rows
	if err != nil {
		fmt.Println(err)
		return
	}

	// Skip the first 5 rows
	rows = rows[5:]

	for i := 0; i < len(rows); i++ {
		opportunityId := rows[i][1]
		reportingStatus := rows[i][2]
		fyQtr := rows[i][3]
		cg := rows[i][4]
		masterClientName := rows[i][5]
		opportunityName := rows[i][6]
		opportunityDetailDescription := rows[i][7]
		industrySegment := rows[i][8]
		stage := rows[i][9]
		totalCurrentRevenue := rows[i][10]
		winProbability := rows[i][11]
		sellingCountry := rows[i][12]
		level2 := rows[i][13]
		salesCapture := rows[i][14]
		createDate := rows[i][15]
		revenue := rows[i][16]

		opportunity := Opportunity{
			OpportunityId:                opportunityId,
			ReportingStatus:              reportingStatus,
			FYQtr:                        fyQtr,
			CG:                           cg,
			MasterClientName:             masterClientName,
			OpportunityName:              opportunityName,
			OpportunityDetailDescription: opportunityDetailDescription,
			IndustrySegment:              industrySegment,
			Stage:                        stage,
			TotalCurrentRevenue:          totalCurrentRevenue,
			WinProbability:               winProbability,
			SellingCountry:               sellingCountry,
			Level2:                       level2,
			SalesCapture:                 salesCapture,
			CreateDate:                   createDate,
			Revenue:                      revenue,
		}

		*opportunities = append(*opportunities, opportunity)
	}
}

func selectPossibleValueById(id string, possibleValues []map[string]interface{}) []map[string]interface{} {
	for _, value := range possibleValues {
		if value["id"] == id {
			var options = value["options"]
			v := reflect.ValueOf(options).Interface().([]interface{})

			var returnArray []map[string]interface{}

			for _, item := range v {
				returnArray = append(returnArray, item.(map[string]interface{}))
			}

			return returnArray
			// return value["options"].([]Item)
			// } else {
			// 	fmt.Println("Unable to convert options to []Item")
			// 	fmt.Println("option: ", value["options"])
			// 	return nil
			// }
		}
	}

	fmt.Println("No possible value found for id: " + id)

	return nil
}

func searchForNearestPossibleOption(value string, options []map[string]interface{}) string {
	for _, possibleValue := range options {
		if strings.Contains(strings.ToLower(possibleValue["value"].(string)), strings.ToLower(value)) {
			return possibleValue["id"].(string)
		}
	}

	return ""
}

func searchForNearestPossibleOptionAndModifyIfItDoesntExist(value string, options []map[string]interface{}) []map[string]interface{} {
	newOptions := options
	found := searchForNearestPossibleOption(value, newOptions)

	if found == "" {
		// If we don't find a match, we create a new option
		newOption := map[string]interface{}{
			"id":    mmModel.NewId(),
			"value": value,
			"color": "propColorDefault",
		}

		newOptions = append(newOptions, newOption)
	}

	return newOptions
}

func convertDateStringToDateObject(dateString string) string {
	if dateString != "" {
		// Truncate the extra precision
		if len(dateString) > 30 {
			dateString = dateString[:30]
		}

		dateString = dateString + "Z"

		// Parse the date string
		t, err := time.Parse(time.RFC3339Nano, dateString)
		if err != nil {
			fmt.Println(err)
		}

		// Convert to epoch timestamp
		epoch := t.UnixNano() / int64(time.Millisecond)
		var dateObject = "{\"from\":" + strconv.FormatInt(epoch, 10) + "}"

		// structure for output: "{\"from\":1706788800000}"
		// "{\"from\":1706788800000}"
		return dateObject
	}

	return ""
}

func convertOpportunityToBlock(opportunity Opportunity, block *model.Block, possibleValues []map[string]interface{}) {
	// block := model.Block{
	// 	Title:     opportunity.OpportunityName,
	// 	BoardID:   boardID,
	// 	Type:      "card",
	// 	CreatedBy: userID,
	// 	ParentID:  "bjz11k7xjopb8fyaqppzog3uuqr",
	// }

	stage := searchForNearestPossibleOption(opportunity.Stage, selectPossibleValueById(stageKey, possibleValues))
	cg := searchForNearestPossibleOption(opportunity.CG, selectPossibleValueById(cgKey, possibleValues))
	date := convertDateStringToDateObject(opportunity.CreateDate)

	// fmt.Println("Stage original value: ", opportunity.Stage)
	// fmt.Println("Stage for ", opportunity.OpportunityName, ": ", stage)
	// fmt.Println("CG original value: ", opportunity.CG)
	// fmt.Println("CG for ", opportunity.OpportunityName, ": ", cg)
	// fmt.Println("Date original value: ", opportunity.CreateDate)
	// fmt.Println("Date for ", opportunity.OpportunityName, ": ", date)
	// fmt.Println("MasterClientName original value: ", opportunity.MasterClientName)
	// fmt.Println("MasterClientName for ", opportunity.OpportunityName, ": ", masterClientName)

	block.Title = opportunity.OpportunityName
	block.Fields["isTemplate"] = false
	// TODO rework this so old information doesn't get overwritten
	if _, ok := block.Fields["properties"]; ok {
		// "properties" exists, so we cast it to the appropriate type and add the new values
		properties := block.Fields["properties"].(map[string]interface{})
		properties[opportunityIdKey] = opportunity.OpportunityId
		properties[stageKey] = stage
		properties[masterClientNameKey] = opportunity.MasterClientName
		properties[totalCurrentRevenueKey] = opportunity.TotalCurrentRevenue
		properties[winProbabilityKey] = opportunity.WinProbability
		properties[salesCaptureKey] = opportunity.SalesCapture
		properties[cgKey] = cg
		properties[level2Key] = opportunity.Level2
		properties[createDateKey] = date
	} else {
		// "properties" does not exist, so we create a new map and add the values to it
		block.Fields["properties"] = map[string]interface{}{
			opportunityIdKey:       opportunity.OpportunityId,
			stageKey:               stage,
			masterClientNameKey:    opportunity.MasterClientName,
			totalCurrentRevenueKey: opportunity.TotalCurrentRevenue,
			winProbabilityKey:      opportunity.WinProbability,
			salesCaptureKey:        opportunity.SalesCapture,
			cgKey:                  cg,
			level2Key:              opportunity.Level2,
			createDateKey:          date,
		}
	}
}

// option:  [
// 	map[color:propColorGreen id:akj61wc9yxdwyw3t6m8igyf9d5o value:Stage 0]
// 	map[color:propColorGray id:aic89a5xox4wbppi6mbyx6ujsda value:Stage 1]
// 	map[color:propColorOrange id:ah6ehh43rwj88jy4awensin8pcw value:Stage 2a]
// 	map[color:propColorRed id:aprhd96zwi34o9cs4xyr3o9sf3c value:Stage 2b]
// 	map[color:propColorPurple id:axesd74yuxtbmw1sbk8ufax7z3a value:WON 🏆]
// 	map[color:propColorDefault id:a5txuiubumsmrs8gsd5jz5gc1oa value:LOST]
// 	map[color:propColorBrown id:acm9q494bcthyoqzmfogxxy5czy value:REQUEST]
// 	map[color:propColorDefault id:a6n6aqi7k65tiqtr5gdzfbosmcw value:Tigital]
// 	map[color:propColorDefault id:ai6u8x1t9fq1mckj8ph1tgfn5uw value:EXT]
// 	map[color:propColorDefault id:aqh8bu8c4ai5broqqnf45ufm5kh value:Stage 3A]
// 	map[color:propColorDefault id:afx6sqzc5q5mmnrs6wpzgx3fwca value:Stage 3B]
// 	map[color:propColorDefault id:af47cinzyk96pufqhsjg84zjo6c value:Stage 3C]
// ]
