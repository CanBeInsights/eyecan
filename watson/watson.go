package watson

import (
	"encoding/json"
	"fmt"
	"github.com/IBM/go-sdk-core/core"
	//"github.com/gosuri/uiprogress"

	//"github.com/gosuri/uiprogress"

	"github.com/watson-developer-cloud/go-sdk/naturallanguageunderstandingv1"
	"math"
	"strings"
	"sync"
	"time"
)

const LOOKUP_TIMEOUT = 5 * time.Second
var nlu *naturallanguageunderstandingv1.NaturalLanguageUnderstandingV1

type historyElem struct {
	ID				string	`json:"id"`
	LastVisitTime	float64	`json:"lastVisitTime"`
	Title			string	`json:"title"`
	TypedCount		int		`json:"typedCount"`
	URL				string	`json:"url"`
	VisitCount		int		`json:"visitCount"`
}

type historyObj struct {
	HistoryElems	[]historyElem      `json:"historyElems"`
}

type categoryElem struct {
	Label			string	`json:"label"`
	Score			float64	`json:"score"`
}

type categoryObj struct {
	CategoryElems	[]categoryElem		`json:"categoryElems"`
}

type historyIntermediate struct {
	ID				string
	LastVisitTime	float64
	Title			string
	TypedCount		int
	URL				string
	VisitCount		int
	Categories		[]naturallanguageunderstandingv1.CategoriesResult
}

type categoryInstance struct {
	ID				string
	LastVisitTime	float64
	TypedCount		int
	VisitCount		int
	InstanceScore	float64
}

type categoryIntermediate struct {
	Label			string
	Instances		[]categoryInstance
}

type historiesIntermediate []historyIntermediate

//"id": "20",
//"lastVisitTime": 1554596357082.7078,
//"title": "javascript - Chrome extension form saving error - Stack Overflow",
//"typedCount": 0,
//"url": "https://stackoverflow.com/questions/36664868/chrome-extension-form-saving-error",
//"visitCount": 1


func init() {
	naturalLanguageUnderstanding, naturalLanguageUnderstandingErr := naturallanguageunderstandingv1.
		NewNaturalLanguageUnderstandingV1(&naturallanguageunderstandingv1.NaturalLanguageUnderstandingV1Options{
			URL: "https://gateway.watsonplatform.net/natural-language-understanding/api",
			Version: "2018-11-16",
			IAMApiKey: "i9jGj42nqgiG6REz6Xn92eFi92Bx6ivVXW2yPLSqkKL9",
		})

	if naturalLanguageUnderstandingErr != nil {
		panic(naturalLanguageUnderstandingErr)
	}

	nlu = naturalLanguageUnderstanding
}

func fetchFromText(text string) *core.DetailedResponse {
	response, responseErr := nlu.Analyze(
		&naturallanguageunderstandingv1.AnalyzeOptions{
			Text: &text,
			Features: &naturallanguageunderstandingv1.Features{
				//Concepts: &naturallanguageunderstandingv1.ConceptsOptions{
				//	Limit: core.Int64Ptr(10),
				//},
				Categories: &naturallanguageunderstandingv1.CategoriesOptions{
					Limit: core.Int64Ptr(10),
				},
			},
		},
	)
	if responseErr != nil {
		panic(responseErr)
	}

	return response
}

func fetchFromURL(url string) (*core.DetailedResponse, error) {
	response, responseErr := nlu.Analyze(
		&naturallanguageunderstandingv1.AnalyzeOptions{
			URL: &url,
			Features: &naturallanguageunderstandingv1.Features{
				//Concepts: &naturallanguageunderstandingv1.ConceptsOptions{
				//	Limit: core.Int64Ptr(10),
				//},
				Categories: &naturallanguageunderstandingv1.CategoriesOptions{
					Limit: core.Int64Ptr(10),
				},
			},
			Language: core.StringPtr("en"),
		},
	)
	if responseErr != nil {
		return nil, responseErr
	}

	return response, nil
}

func Lookup(url string) (string, error) {
	response, err := fetchFromURL(url)
	if err != nil {
		return "", err
	}
	result := nlu.GetAnalyzeResult(response)
	b, _ := json.MarshalIndent(result, "", "  ")
	return string(b), nil
}

func LookupCategories(url string) (string, error) {
	response, err0 := fetchFromURL(url)
	if err0 != nil {
		return "", err0
	}
	result := nlu.GetAnalyzeResult(response).Categories

	b, err1 := json.MarshalIndent(result[0], "", "	")
	if err1 != nil {
		return "", err1
	}
	return string(b), nil

}

func UnmarshalURLs(urlString string) (historyObj, error) {
	quoteStripped := strings.Trim(urlString, "\"")
	deslashed := strings.Replace(quoteStripped, `\"`, `"`, 100000000000)
	inputJSON := `{ "historyElems": ` + deslashed + ` }`	// Wrap array in valid outer JSON matching marshall struct
	fmt.Println(inputJSON)
	var res historyObj
	// TODO: Handle this unhandled error, possibly using the code below it
	err := json.Unmarshal([]byte(inputJSON), &res)
	if err != nil {
		return historyObj{}, err
	}
	return res, nil
}

// ToElems function receiver compiles the final set of output data based on the input data
func (hs historiesIntermediate) ToElems() []categoryElem {
	// TODO: Perfect this, as this is really where the score determination magic happens

	// Step One: Combine each category into an overall `[]categoryIntermediate` categories object
	var cats []categoryIntermediate
	for _, h := range hs {
		// Iterate over the categories so the data from this URL visit can be applied to all associated categories
		for _, newCat := range h.Categories {
			exists := false
			// If the label is already in `cats`, add it to that section and move on to the next new category
			for i, cat := range cats {
				if *newCat.Label == cat.Label {
					cats[i].Instances = append(cats[i].Instances, categoryInstance{
						ID:            h.ID,
						LastVisitTime: h.LastVisitTime,
						TypedCount:    h.TypedCount,
						VisitCount:    h.VisitCount,
						InstanceScore: *newCat.Score,	// Note just copying the score for now
					})
					exists = true
					break;
				}
			}

			if !exists {
				var instances = make([]categoryInstance, 1)
				instances[0] = categoryInstance{
					ID:            h.ID,
					LastVisitTime: h.LastVisitTime,
					TypedCount:    h.TypedCount,
					VisitCount:    h.VisitCount,
					InstanceScore: *newCat.Score,	// Note just copying the score for now
				}
				cats = append(cats, categoryIntermediate{
					Label: *newCat.Label,
					Instances: instances,
				})
			}
		}
	}

	// Step Two: Calculate the overall score from each `categoryIntermediate` object to create a `categoryObj`
	var catObj categoryObj
	//var visits categoryObj
	for _, cat := range cats {
		// This here is the secret sauce, the process of devising a score given visit data
		// The current way this works is by scaling up/down the score given based on a number of "penalties"
		//visits.CategoryElems = append(visits.CategoryElems, categoryElem{
		//	Label: cat.Label,
		//})
		//visits.CategoryElems[i].Label = cat.Label		// TODO: Remove

		totalScore := 0.0
		totalPenalty := 0.0
		for _, instance := range cat.Instances {
			// Penalty 1: Increase score with more visits, over a base of 1000 which also acts as max readable visits
			numerator := math.Min(float64(instance.VisitCount), 1000.0)
			denominator := 1000.0

			//visits.CategoryElems[i].Score += float64(instance.VisitCount) // TODO: Remove

			// Penalty 2: Keep score if at most 14 days old last visit, otherwise divide by number of 14-day periods
			timeAgo := time.Duration(time.Now().Unix() * 1000 - int64(instance.LastVisitTime)) * time.Millisecond
			if timeAgo > 14 * 24 * time.Hour {
				denominator = denominator * float64(timeAgo / 14 * 24 * time.Hour)
			}

			totalScore += instance.InstanceScore
			totalPenalty += numerator / denominator
		}

		catObj.CategoryElems = append(catObj.CategoryElems, categoryElem{
			Label: cat.Label,
			Score: totalScore / float64(len(cat.Instances)) * totalPenalty,
		})
	}

	//b, _ := json.MarshalIndent(catObj.CategoryElems, "", "	")
	//fmt.Println(string(b))
	return catObj.CategoryElems
}

func LookupsCategories(urlString string) (string, error) {
	var c = make(chan historyIntermediate)
	var e = make(chan error)
	unmarshalled, err := UnmarshalURLs(urlString)
	if err != nil {
		return "", err
	}
	elems := unmarshalled.HistoryElems
	var wg sync.WaitGroup

	// Send off each URL to its own goroutine as explained below
	wg.Add(len(elems))

	// Create progress bar
	//uiprogress.Start()            // start rendering
	//bar := uiprogress.AddBar(len(elems)) // Add a new bar
	//bar.AppendCompleted()
	//bar.PrependElapsed()

	for _, elem := range elems {
		// Each URL is tested using a different goroutine, with the `response` objects sent back in the `c` channel
		go func(c chan<- historyIntermediate, e chan<- error, elem historyElem) {
			defer wg.Done()

			response, err := fetchFromURL(elem.URL)			// TODO: Keep in mind an error is being discarded here
			if err != nil {
				e <- err
				return					// Because `response` is likely not correct on error, return before using it
			}

			categories := nlu.GetAnalyzeResult(response).Categories
			h := historyIntermediate{
				ID:				elem.ID,
				LastVisitTime:	elem.LastVisitTime,
				Title:			elem.Title,
				TypedCount:		elem.TypedCount,
				URL:			elem.URL,
				VisitCount:		elem.VisitCount,
				Categories:		categories,
			}

			//bar.Incr()
			c <- h
		}(c, e, elem)			// Note `elem` being included to prevent problems with progression of `elem` as in JS
	}

	// Close the channel when all goroutines are done, in a goroutine so the `wg.Wait()` doesn't block the main thread
	go func() {
		wg.Wait()
		close(c)
	}()

	// Alternatively, quit early without closing the channel for a quick and dirty timeout TODO: Fix This Memory Leak
	timeout := false

	//Collect results
	//var res []categoryElem
	var hs historiesIntermediate

	for {
		select {
			case <-time.After(1 * time.Second):
				timeout = true
				break
			case response, open := <-c:
				if open {
					hs = append(hs, response)
				} else {
					timeout = true
					break
				}
		}
		if timeout {
			break
		}
	}

	//bar.Set(len(elems))

	//for response := range c {		// Note how in the error handling above, `c` only receives completed results
	//	hs = append(hs, response)
	//}

	outputElems := hs.ToElems()

	////TODO: Note we do technically have an `hs` with only completed and thus valid results, but we'll `err` anyway
	//select {
	//// In case of error back in the requesting step, return `err` now
	//case err := <-e:
	//	return "", err
	//// In case of no error back in the requesting step, continue
	//default:
	//	// Do nothing here, so it continues on. The default just prevents blocking as a try-receive
	//}

	b, jsonErr := json.MarshalIndent(outputElems, "", "	")
	if jsonErr != nil {
		return "", jsonErr
	}

	return string(b), nil
}

func Test() {
	text := `IBM is an American multinational technology company
         headquartered in Armonk, New York, United States
         with operations in over 170 countries.`
	response, responseErr := nlu.Analyze(
		&naturallanguageunderstandingv1.AnalyzeOptions{
			Text: &text,
			Features: &naturallanguageunderstandingv1.Features{
				//Concepts: &naturallanguageunderstandingv1.ConceptsOptions{
				//	Limit: core.Int64Ptr(10),
				//},
				Categories: &naturallanguageunderstandingv1.CategoriesOptions{
					Limit: core.Int64Ptr(10),
				},
			},
		},
	)
	if responseErr != nil {
		panic(responseErr)
	}
	result := nlu.GetAnalyzeResult(response)
	b, _ := json.MarshalIndent(result, "", "	")
	fmt.Println(string(b))
}