package lctrn

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf16"
)

type popupMethod struct {
	opsys string
	tool  string
	fn    func(title, msg string) error
}

var popupMethods = []popupMethod{
	{
		opsys: "darwin",
		tool:  "osascript",
		fn: func(title, msg string) error {
			title = strings.ReplaceAll(title, `"`, `\"`)
			msg = strings.ReplaceAll(msg, `"`, `\"`)
			command := fmt.Sprintf(`display dialog "%v" with title "%v" buttons {"OK"} with icon stop`, msg, title)
			return exec.Command("osascript", "-e", command).Run()
		},
	},
	{
		opsys: "windows",
		tool:  "powershell",
		fn: func(title, msg string) error {
			title = strings.ReplaceAll(title, "'", "''")
			msg = strings.ReplaceAll(msg, "'", "''")
			command := fmt.Sprintf("Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.MessageBox]::Show('%v','%v',0,16)", msg, title)
			command = encodeUTF16LE(command)
			return exec.Command("powershell", "-NoProfile", "-EncodedCommand", command).Run()
		},
	},
	{
		opsys: "",
		tool:  "zenity",
		fn: func(title, msg string) error {
			return exec.Command("zenity", "--error", "--title="+title, "--text="+msg).Run()
		},
	},
	{
		opsys: "",
		tool:  "kdialog",
		fn: func(title, msg string) error {
			return exec.Command("kdialog", "--error", msg, "--title", title).Run()
		},
	},
	{
		opsys: "",
		tool:  "xmessage",
		fn: func(title, msg string) error {
			msg = fmt.Sprintf("%v\n\n%v", title, msg)
			return exec.Command("xmessage", "-center", "-geometry", "400x200", msg).Run()
		},
	},

	// Text file fallbacks
	{
		opsys: "darwin",
		tool:  "open",
		fn: func(title, msg string) error {
			f, err := createTempFile(title, msg)
			if err != nil {
				return err
			}
			return exec.Command("open", f).Run()
		},
	},
	{
		opsys: "",
		tool:  "xdg-open",
		fn: func(title, msg string) error {
			f, err := createTempFile(title, msg)
			if err != nil {
				return err
			}
			return exec.Command("xdg-open", f).Run()
		},
	},
	{
		opsys: "windows",
		tool:  "cmd",
		fn: func(title, msg string) error {
			f, err := createTempFile(title, msg)
			if err != nil {
				return err
			}
			return exec.Command("cmd", "/c", "start", f).Run()
		},
	},
}

func createTempFile(title, msg string) (string, error) {
	f, err := os.CreateTemp("", "popup__*.txt")
	if err != nil {
		return "", err
	}

	fmt.Fprintf(f, "%v\n\n%v", title, msg)

	if err := f.Sync(); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	return f.Name(), err
}

func encodeUTF16LE(s string) string {
	u16 := utf16.Encode([]rune(s))
	data := []byte{}
	for _, r := range u16 {
		data = append(data, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(data)
}

func (app *App) configurePopup() bool {
	for _, method := range popupMethods {
		if method.opsys != "" && method.opsys != runtime.GOOS {
			continue
		}

		_, err := exec.LookPath(method.tool)
		if err == nil {
			app.popupFunc = method.fn
			return true
		}
	}

	return false
}

func (app *App) Popup(title, msg string) {
	if app.popupFunc == nil {
		if !app.configurePopup() {
			fmt.Fprintf(os.Stderr, "[%v] %v\n", title, msg)
			return
		}
	}

	err := app.popupFunc(title, msg)

	if err != nil {
		fmt.Fprintf(os.Stderr, "[%v] %v\n", title, msg)
		log.Println(fmt.Errorf("failed to show popup: %w", err))
	}
}
