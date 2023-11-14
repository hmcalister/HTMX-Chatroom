package api

import "sync"

type ApplicationState struct {
	totalMessagesSent int
	appStateMutex     sync.Mutex
}

func NewApplicationState() *ApplicationState {
	return &ApplicationState{
		totalMessagesSent: 0,
	}
}

func (appState *ApplicationState) NewMessage() int {
	appState.appStateMutex.Lock()
	defer appState.appStateMutex.Unlock()
	appState.totalMessagesSent += 1
	return appState.totalMessagesSent
}
