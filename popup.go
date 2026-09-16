package lctrn

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"unicode/utf16"
)

type popupMethod struct {
	name     string
	callback func(title, msg string) error
}

var popupMethods = []popupMethod{
	{
		name: "osascript",
		callback: func(title, msg string) error {
			return exec.Command("osascript", "-e",
				fmt.Sprintf(`display dialog "%v" with title "%v" buttons {"OK"} with icon stop`, msg, title),
			).Run()
		},
	},
	{
		name: "powershell",
		callback: func(title, msg string) error {
			command := fmt.Sprintf("Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show('%v','%v',0,16)", msg, title)
			command = encodeUTF16LE(command)
			return exec.Command("powershell", "-NoProfile", "-EncodedCommand", command).Run()
		},
	},
	{
		name: "zenity",
		callback: func(title, msg string) error {
			return exec.Command("zenity", "--error", "--title="+title, "--text="+msg).Run()
		},
	},
	{
		name: "kdialog",
		callback: func(title, msg string) error {
			return exec.Command("kdialog", "--error", msg, "--title", title).Run()
		},
	},
	{
		name: "xmessage",
		callback: func(title, msg string) error {
			return exec.Command("xmessage", "-center", msg).Run()
		},
	},
}

func encodeUTF16LE(s string) string {
	u16 := utf16.Encode([]rune(s))
	data := []byte{}
	for _, r := range u16 {
		data = append(data, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(data)
}

func popupUsingTxtFile(title, msg string) error {
	f, err := os.CreateTemp("", fmt.Sprintf("%v__*.txt", title))
	if err != nil {
		return err
	}

	if _, err := f.WriteString(msg); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return exec.Command("open", f.Name()).Run()
}

func (app *App) configurePopup() {
	if app.popupFunc != nil {
		return
	}

	for _, method := range popupMethods {
		_, err := exec.LookPath(method.name)
		if err == nil {
			app.popupFunc = method.callback
			return
		}
	}

	app.popupFunc = popupUsingTxtFile
}

func (app *App) Popup(title, msg string) {
	if app.popupFunc == nil {
		app.configurePopup()
	}

	err := app.popupFunc(title, msg)

	if err != nil {
		log.Println(fmt.Errorf("failed to show popup: %w", err))
	}
}
