package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dgnsrekt/goconvey-notify/web/server/contract"
	"github.com/dgnsrekt/goconvey-notify/web/server/messaging"
)

type NotificationConfig struct {
	Sound struct {
		FilePath        string `json:"file_path"`        // Legacy single file
		SuccessFilePath string `json:"success_file_path"` // New: success sound
		FailureFilePath string `json:"failure_file_path"` // New: failure sound
	} `json:"sound"`
	NTFY struct {
		Server     string `json:"server"`
		Topic      string `json:"topic"`
		Timeout    int    `json:"timeout"`
		AuthHeader string `json:"auth_header"`
	} `json:"ntfy"`
}

type HTTPServer struct {
	watcher     chan messaging.WatcherCommand
	executor    contract.Executor
	latest      *contract.CompleteOutput
	currentRoot string
	longpoll    chan chan string
	paused      bool
	config      *NotificationConfig
}

func (self *HTTPServer) ReceiveUpdate(root string, update *contract.CompleteOutput) {
	self.currentRoot = root
	self.latest = update
}

func (self *HTTPServer) isValidSoundFile(path string) bool {
	if path == "" {
		return false
	}

	// Check file exists and is not a directory
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}

	// Check extension for common web audio formats
	ext := strings.ToLower(filepath.Ext(path))
	validExts := []string{".mp3", ".wav", ".ogg", ".m4a", ".webm"}
	for _, valid := range validExts {
		if ext == valid {
			return true
		}
	}

	return false
}

func (self *HTTPServer) isValidNTFYServer(server string) bool {
	if server == "" {
		return false
	}

	u, err := url.Parse(server)
	if err != nil {
		return false
	}

	// Must be http or https
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Must have a host
	if u.Host == "" {
		return false
	}

	return true
}

func (self *HTTPServer) isValidNTFYTopic(topic string) bool {
	if topic == "" {
		return false
	}

	// Valid characters: A-Z, a-z, 0-9, _, -
	validRegex := regexp.MustCompile("^[A-Za-z0-9_-]+$")
	return validRegex.MatchString(topic)
}

func (self *HTTPServer) isSuccessSoundEnabled() bool {
	// Check new success file first, then fall back to legacy file
	if self.config.Sound.SuccessFilePath != "" {
		return self.isValidSoundFile(self.config.Sound.SuccessFilePath)
	}
	return self.isValidSoundFile(self.config.Sound.FilePath)
}

func (self *HTTPServer) isFailureSoundEnabled() bool {
	// Check new failure file first, then fall back to legacy file
	if self.config.Sound.FailureFilePath != "" {
		return self.isValidSoundFile(self.config.Sound.FailureFilePath)
	}
	return self.isValidSoundFile(self.config.Sound.FilePath)
}

func (self *HTTPServer) isSoundEnabled() bool {
	return self.isSuccessSoundEnabled() || self.isFailureSoundEnabled()
}

func (self *HTTPServer) isNTFYEnabled() bool {
	return self.isValidNTFYServer(self.config.NTFY.Server) && self.isValidNTFYTopic(self.config.NTFY.Topic)
}

func (self *HTTPServer) Watch(response http.ResponseWriter, request *http.Request) {
	if request.Method == "POST" {
		self.adjustRoot(response, request)
	} else if request.Method == "GET" {
		response.Write([]byte(self.currentRoot))
	}
}

func (self *HTTPServer) adjustRoot(response http.ResponseWriter, request *http.Request) {
	newRoot := self.parseQueryString("root", response, request)
	if newRoot == "" {
		return
	}
	info, err := os.Stat(newRoot) // TODO: how to unit test?
	if !info.IsDir() || err != nil {
		http.Error(response, err.Error(), http.StatusNotFound)
		return
	}

	self.watcher <- messaging.WatcherCommand{
		Instruction: messaging.WatcherAdjustRoot,
		Details:     newRoot,
	}
}

func (self *HTTPServer) Ignore(response http.ResponseWriter, request *http.Request) {
	paths := self.parseQueryString("paths", response, request)
	if paths != "" {
		self.watcher <- messaging.WatcherCommand{
			Instruction: messaging.WatcherIgnore,
			Details:     paths,
		}
	}
}

func (self *HTTPServer) Reinstate(response http.ResponseWriter, request *http.Request) {
	paths := self.parseQueryString("paths", response, request)
	if paths != "" {
		self.watcher <- messaging.WatcherCommand{
			Instruction: messaging.WatcherReinstate,
			Details:     paths,
		}
	}
}

func (self *HTTPServer) parseQueryString(key string, response http.ResponseWriter, request *http.Request) string {
	value := request.URL.Query()[key]

	if len(value) == 0 {
		http.Error(response, fmt.Sprintf("No '%s' query string parameter included!", key), http.StatusBadRequest)
		return ""
	}

	path := value[0]
	if path == "" {
		http.Error(response, "You must provide a non-blank path.", http.StatusBadRequest)
	}
	return path
}

func (self *HTTPServer) Status(response http.ResponseWriter, request *http.Request) {
	status := self.executor.Status()
	response.Write([]byte(status))
}

func (self *HTTPServer) LongPollStatus(response http.ResponseWriter, request *http.Request) {
	if self.executor.ClearStatusFlag() {
		response.Write([]byte(self.executor.Status()))
		return
	}

	timeout, err := strconv.Atoi(request.URL.Query().Get("timeout"))
	if err != nil || timeout > 180000 || timeout < 0 {
		timeout = 60000 // default timeout is 60 seconds
	}

	myReqChan := make(chan string)

	select {
	case self.longpoll <- myReqChan: // this case means the executor's status is changing
	case <-time.After(time.Duration(timeout) * time.Millisecond): // this case means the executor hasn't changed status
		return
	}

	out := <-myReqChan

	if out != "" { // TODO: Why is this check necessary? Sometimes it writes empty string...
		response.Write([]byte(out))
	}
}

func (self *HTTPServer) Results(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	response.Header().Set("Pragma", "no-cache")
	response.Header().Set("Expires", "0")
	if self.latest != nil {
		self.latest.Paused = self.paused
	}
	stuff, _ := json.Marshal(self.latest)
	response.Write(stuff)
}

func (self *HTTPServer) Execute(response http.ResponseWriter, request *http.Request) {
	go self.execute()
}

func (self *HTTPServer) execute() {
	self.watcher <- messaging.WatcherCommand{Instruction: messaging.WatcherExecute}
}

func (self *HTTPServer) TogglePause(response http.ResponseWriter, request *http.Request) {
	instruction := messaging.WatcherPause
	if self.paused {
		instruction = messaging.WatcherResume
	}

	self.watcher <- messaging.WatcherCommand{Instruction: instruction}
	self.paused = !self.paused

	fmt.Fprint(response, self.paused) // we could write out whatever helps keep the UI honest...
}

func (self *HTTPServer) SendNTFY(response http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		http.Error(response, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	title := request.FormValue("title")
	body := request.FormValue("body")

	if title == "" || body == "" {
		http.Error(response, "Missing title or body", http.StatusBadRequest)
		return
	}

	// Load current config and send notification
	self.sendNTFYNotification(self.config, title, body)
	response.WriteHeader(http.StatusOK)
}

func (self *HTTPServer) ConfigStatus(response http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		http.Error(response, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response.Header().Set("Content-Type", "application/json")

	status := map[string]interface{}{
		"soundConfigured":        self.isSoundEnabled(),
		"successSoundConfigured": self.isSuccessSoundEnabled(),
		"failureSoundConfigured": self.isFailureSoundEnabled(),
		"ntfyConfigured":         self.isNTFYEnabled(),
	}

	json.NewEncoder(response).Encode(status)
}

func (self *HTTPServer) SoundFile(response http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		http.Error(response, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if self.config.Sound.FilePath == "" {
		http.Error(response, "No sound file configured", http.StatusNotFound)
		return
	}

	// Check if file exists
	if _, err := os.Stat(self.config.Sound.FilePath); os.IsNotExist(err) {
		http.Error(response, "Sound file not found", http.StatusNotFound)
		return
	}

	http.ServeFile(response, request, self.config.Sound.FilePath)
}

func (self *HTTPServer) SuccessSoundFile(response http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		http.Error(response, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var filePath string
	if self.config.Sound.SuccessFilePath != "" {
		filePath = self.config.Sound.SuccessFilePath
	} else {
		filePath = self.config.Sound.FilePath
	}

	if filePath == "" {
		http.Error(response, "No success sound file configured", http.StatusNotFound)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(response, "Success sound file not found", http.StatusNotFound)
		return
	}

	http.ServeFile(response, request, filePath)
}

func (self *HTTPServer) FailureSoundFile(response http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		http.Error(response, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var filePath string
	if self.config.Sound.FailureFilePath != "" {
		filePath = self.config.Sound.FailureFilePath
	} else {
		filePath = self.config.Sound.FilePath
	}

	if filePath == "" {
		http.Error(response, "No failure sound file configured", http.StatusNotFound)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(response, "Failure sound file not found", http.StatusNotFound)
		return
	}

	http.ServeFile(response, request, filePath)
}

func (self *HTTPServer) sendNTFYNotification(config *NotificationConfig, title, body string) {
	if !self.isNTFYEnabled() {
		return
	}

	client := &http.Client{
		Timeout: time.Duration(config.NTFY.Timeout) * time.Second,
	}

	// NTFY requires the topic in the URL path
	url := config.NTFY.Server + "/" + config.NTFY.Topic
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		log.Printf("NTFY request creation failed: %v", err)
		return
	}

	req.Header.Set("Title", title)
	if config.NTFY.AuthHeader != "" {
		req.Header.Set("Authorization", config.NTFY.AuthHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("NTFY request failed: %v", err)
		return
	}
	resp.Body.Close()
}

func LoadNotificationConfig(path string) (*NotificationConfig, error) {
	config := &NotificationConfig{}

	// Set defaults
	config.Sound.FilePath = ""
	config.Sound.SuccessFilePath = ""
	config.Sound.FailureFilePath = ""
	config.NTFY.Server = "https://ntfy.sh"
	config.NTFY.Topic = "goconvey-notifications"
	config.NTFY.Timeout = 30
	config.NTFY.AuthHeader = ""

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Notification config file not found: %s, using defaults", path)
		return config, nil
	}

	// Read and parse config file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Backward compatibility: if legacy file_path is set but new fields are empty,
	// use the legacy file_path for both success and failure sounds
	if config.Sound.FilePath != "" && config.Sound.SuccessFilePath == "" && config.Sound.FailureFilePath == "" {
		log.Printf("Using legacy sound config - applying '%s' to both success and failure", config.Sound.FilePath)
	}

	log.Printf("Loaded notification config from: %s", path)
	return config, nil
}

func NewHTTPServer(
	root string,
	watcher chan messaging.WatcherCommand,
	executor contract.Executor,
	status chan chan string,
	config *NotificationConfig) *HTTPServer {

	self := new(HTTPServer)
	self.currentRoot = root
	self.watcher = watcher
	self.executor = executor
	self.longpoll = status
	self.config = config
	return self
}
