package requests

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// SendAnalytics send service analytics to http://<serviceAddr>/api/analytics
/*  Json body
{
	"AvgPermanenceTime": float   //expressed in milliseconds
	"EnqueueDequeueRatio": float //<1 dequeue faster than enqueue, ==1 balanced, >1 enqueue faster than dequeue
	"SpaceFull": int          //0-100% value expressing the space lect
    "ThresholdRatio": float     //0-1 value representing (N.Thresholded values/Tot number of deletion)
	"ElementPerSecondIn": float
	"ElementPerSecondOut": float
}
*/
func SendAnalytics(analyticsJson string, serviceAddr string) {
	client := http.Client{
		Timeout: 900 * time.Millisecond,
	}
	url := fmt.Sprintf("http://%s/api/analytics", serviceAddr)
	_, err := client.Post(url, "application/json", strings.NewReader(analyticsJson))
	if err != nil {
		log.Println("ERROR: POST analytics: ", err)
		log.Println("Body: ", analyticsJson)
	}
}
