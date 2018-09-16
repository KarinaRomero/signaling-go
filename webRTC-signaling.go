package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)

type clientWs struct {
	ws   *websocket.Conn
	Name string
}

var clientsWs = []clientWs{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	}}

func main() {

	http.HandleFunc("/", handleConnections)

	fmt.Println("Server listen to port: 8888")
	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}

}

func handleConnections(w http.ResponseWriter, r *http.Request) {

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer ws.Close()

	for {
		var f interface{}

		messageType, p, err := ws.ReadMessage()

		if err != nil {
			log.Printf("error: %v", err)
			fmt.Println(messageType)
			delete(clients, ws)
			break
		}

		json.Unmarshal(p, &f)

		itemsMap := f.(map[string]interface{})
		//fmt.Println(itemsMap)

		switch itemsMap["type"] {

		case "login":

			if searchClient(itemsMap["name"].(string), nil) == -1 {
				var client = clientWs{
					ws:   ws,
					Name: itemsMap["name"].(string),
				}
				clientsWs = append(clientsWs, client)

				//fmt.Println("User logged in as ", itemsMap["name"].(string))

				js, err := json.Marshal(struct {
					Type   string `json:"type"`
					Succes bool   `json:"success"`
				}{
					itemsMap["type"].(string),
					true,
				})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				ws.WriteMessage(1, js)

			} else {
				js, err := json.Marshal(struct {
					Type   string `json:"type"`
					Succes bool   `json:"success"`
				}{
					itemsMap["type"].(string),
					false,
				})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				ws.WriteMessage(1, js)
			}

		case "offer":
			fmt.Println("Send offer to", itemsMap["name"].(string))

			var Otherindex = searchClient(itemsMap["name"].(string), nil)
			var myIndex = searchClient("", ws)

			if myIndex != -1 && Otherindex != -1 {
				type offer struct {
					Type string `json:"type"`
					SDP  string `json:"sdp"`
				}

				var off map[string]interface{}
				off = itemsMap["offer"].(map[string]interface{})

				var offerStructure = offer{
					Type: "offer",
					SDP:  off["sdp"].(string),
				}

				js, err := json.Marshal(struct {
					Type  string `json:"type"`
					Offer offer  `json:"offer"`
					Name  string `json:"name"`
				}{
					"offer",
					offerStructure,
					clientsWs[myIndex].Name,
				})

				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				clientsWs[Otherindex].ws.WriteMessage(1, js)
			}

		case "answer":
			fmt.Println("answer to: ", itemsMap["name"].(string))

			var Otherindex = searchClient(itemsMap["name"].(string), nil)

			if Otherindex != -1 {
				type answer struct {
					Type string `json:"type"`
					SDP  string `json:"sdp"`
				}
				var aws map[string]interface{}
				aws = itemsMap["answer"].(map[string]interface{})

				//fmt.Println(aws["sdp"].(string))

				var answerStructure = answer{
					Type: "answer",
					SDP:  aws["sdp"].(string),
				}

				js, err := json.Marshal(struct {
					Type   string `json:"type"`
					Answer answer `json:"answer"`
				}{
					"answer",
					answerStructure,
				})

				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				clientsWs[Otherindex].ws.WriteMessage(1, js)
			}

		case "candidate":
			fmt.Println("Sending candidate to ", itemsMap["name"].(string))

			var Otherindex = searchClient(itemsMap["name"].(string), nil)

			if Otherindex != -1 {
				type candidate struct {
					Candidate     string  `json:"candidate"`
					SdpMid        string  `json:"sdpMid"`
					SdpMLineIndex float64 `json:"sdpMLineIndex"`
				}
				var cnddt map[string]interface{}
				cnddt = itemsMap["candidate"].(map[string]interface{})

				var candidateToSend = candidate{
					Candidate:     cnddt["candidate"].(string),
					SdpMid:        cnddt["sdpMid"].(string),
					SdpMLineIndex: cnddt["sdpMLineIndex"].(float64),
				}

				js, err := json.Marshal(struct {
					Type      string    `json:"type"`
					Candidate candidate `json:"candidate"`
				}{
					"candidate",
					candidateToSend,
				})

				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				clientsWs[Otherindex].ws.WriteMessage(1, js)

			}

		case "leave":
			//fmt.Println("leave: ", itemsMap["name"].(string))

			var Otherindex = searchClient(itemsMap["name"].(string), nil)

			if Otherindex != -1 {
				js, err := json.Marshal(struct {
					Type string `json:"type"`
				}{
					"leave",
				})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				clientsWs[Otherindex].ws.WriteMessage(1, js)
			}

		default:
			//fmt.Println("default")
			js, err := json.Marshal(struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				"error",
				"Unrecognized command: " + itemsMap["candidate"].(string),
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ws.WriteMessage(1, js)

		}
	}
}

func searchClient(c string, ws *websocket.Conn) int {
	if c == "" {
		for i := range clientsWs {
			if clientsWs[i].ws == ws {
				//found
				return i
			}
		}
	} else {
		for i := range clientsWs {
			if clientsWs[i].Name == c {
				//found
				return i
			}
		}
	}
	//not found
	return -1
}

/*func createKeyValuePairs(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
	}
	return b.String()
}
func main() {
	m := map[string]string{
		"LOG_LEVEL": "DEBUG",
		"API_KEY":   "12345678-1234-1234-1234-1234-123456789abc",
	}
	println(createKeyValuePairs(m))

}*/
