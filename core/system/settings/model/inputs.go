package model

type SettingListInput struct{}

type SettingKeyInput struct {
	Key string `input:"arguments.key"`
}

type SetSettingInput struct {
	Key   string `input:"arguments.key"`
	Value any    `input:"payload.value"`
}