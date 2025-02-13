package screens

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/fairdatasociety/fairpass/internal/utils"
	"github.com/google/uuid"
	generator "github.com/sethvargo/go-password/password"
)

func (main *mainView) makeAddPasswordView(i *password) fyne.CanvasObject {
	if i == nil {
		i = &password{}
	}
	websiteEntry := widget.NewEntryWithData(binding.BindString(&i.Domain))
	websiteEntry.SetPlaceHolder("Domain Name")
	websiteEntry.Validator = func(s string) error {
		if s == "" {
			return fmt.Errorf("please enter a domain name")
		}
		return nil
	}

	usernameEntry := widget.NewEntryWithData(binding.BindString(&i.Username))
	usernameEntry.SetPlaceHolder("Username")
	usernameEntry.Validator = func(s string) error {
		if s == "" {
			return fmt.Errorf("please enter a username")
		}
		return nil
	}

	noteEntry := widget.NewEntryWithData(binding.BindString(&i.Description))
	noteEntry.SetPlaceHolder("Description")
	noteEntry.MultiLine = true
	noteEntry.Validator = nil

	passwordEntry := widget.NewPasswordEntry()
	var passwordBind binding.ExternalString

	if i.Password == "" {
		passwordBind = binding.BindString(&i.Password)
	} else {
		var err error
		password, err := main.index.encryptor.DecryptContent(main.index.password, i.Password)
		if err != nil {
			fmt.Println("failed to decrypt password ", err)
		}
		passwordBind = binding.BindString(&password)
	}

	passwordEntry.Bind(passwordBind)
	passwordEntry.Validator = func(s string) error {
		if s == "" {
			return fmt.Errorf("please enter a password")
		}
		return nil
	}
	passwordEntry.SetPlaceHolder("Password")
	passwordCopyButton := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		main.index.Window.Clipboard().SetContent(passwordEntry.Text)
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "FairPass",
			Content: "Password copied to clipboard",
		})
	})
	g := &generatorOptions{
		Length:       16,
		NumDigits:    0,
		NumSymbols:   0,
		NoUpper:      false,
		AllowRepeat:  true,
		AllowSymbols: true,
		AllowDigits:  true,
	}
	passwordMakeButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		d := dialog.NewCustomConfirm("Generate password", "Ok", "Cancel", i.makePasswordDialog(g), func(b bool) {
			if b {
				passwordBind.Set(i.Password)
			}
		}, main.index.Window)
		d.Show()
	})
	var passwordObject fyne.CanvasObject
	if i.ID != "" {
		websiteEntry.Disable()
		usernameEntry.Disable()
		passwordEntry.Disable()
		noteEntry.Disable()
		passwordObject = container.NewBorder(nil, nil, nil, container.NewHBox(passwordCopyButton), passwordEntry)
	} else {
		passwordObject = container.NewBorder(nil, nil, nil, container.NewHBox(passwordCopyButton, passwordMakeButton), passwordEntry)
	}

	form := container.NewVBox(
		labelWithStyle("Domain"),
		websiteEntry,
		labelWithStyle("Username"),
		usernameEntry,
		labelWithStyle("Password"),
		passwordObject,
		labelWithStyle("Description"),
		noteEntry,
	)

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		passwordsView := newListView(main)
		main.setContent(passwordsView.view)
	})

	var top fyne.CanvasObject
	if i.ID == "" {
		saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
			main.index.progress = dialog.NewProgressInfinite("", "Saving Password", main.index) //lint:ignore SA1019 fyne-io/fyne/issues/2782
			main.index.progress.Show()
			defer main.index.progress.Hide()
			if i.Domain == "" {
				return
			}
			if i.Username == "" {
				return
			}
			if i.Password == "" {
				return
			}
			i.ID = uuid.New().String()

			var err error
			i.Password, err = main.index.encryptor.EncryptContent(main.index.password, i.Password)
			if err != nil {
				fmt.Println("failed to save password")
				return
			}
			i.GeneratorOptions = g
			i.CreatedAt = time.Now().Unix()
			i.UpdatedAt = time.Now().Unix()
			i.IsStarred = "false"
			data, err := json.Marshal(i)
			if err != nil {
				fmt.Println("failed to save password")
				return
			}
			err = main.index.dfsAPI.DocPut(main.index.sessionID, utils.PodName, utils.PasswordsTable, data)
			if err != nil {
				fmt.Println("failed to save password")
				return
			}
			passwordsView := newListView(main)
			main.setContent(passwordsView.view)
		})
		saveBtn.Importance = widget.HighImportance
		top = container.NewBorder(nil, nil, cancelBtn, saveBtn, widget.NewLabel(""))
	} else {
		top = container.NewBorder(nil, nil, cancelBtn, nil, widget.NewLabel(""))
	}

	return container.NewPadded(container.NewBorder(top, nil, nil, nil, form))
}

func (i *password) makePasswordDialog(g *generatorOptions) fyne.CanvasObject {
	passwordBind := binding.BindString(&i.Password)
	passwordEntry := widget.NewEntryWithData(passwordBind)
	passwordEntry.Validator = nil
	refreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		err := makePassword(g, passwordBind)
		if err != nil {
			return
		}
	})

	lengthBind := binding.BindInt(&g.Length)
	if g.Length == 0 {
		lengthBind.Set(16)
	}

	lengthEntry := widget.NewEntryWithData(binding.IntToString(lengthBind))
	lengthEntry.Disabled()
	lengthEntry.Validator = nil
	lengthEntry.OnChanged = func(s string) {
		if s == "" {
			return
		}
		l, err := strconv.Atoi(s)
		if err != nil {
			log.Println(err)
			return
		}
		if l < 8 || l > 128 {
			log.Printf("password lenght must be between %d and %d, got %d", 8, 128, l)
			return
		}
		lengthBind.Set(l)
		err = makePassword(g, passwordBind)
		if err != nil {
			return
		}
	}

	lengthSlider := widget.NewSlider(float64(8), float64(128))
	lengthSlider.OnChanged = func(f float64) {
		lengthBind.Set(int(f))
		err := makePassword(g, passwordBind)
		if err != nil {
			return
		}
	}

	digitsButton := widget.NewCheck("0-9", func(isChecked bool) {
		g.AllowDigits = isChecked
		err := makePassword(g, passwordBind)
		if err != nil {
			return
		}
	})
	digitsButton.SetChecked(true)

	symbolsButton := widget.NewCheck("!%$", func(isChecked bool) {
		g.AllowSymbols = isChecked
		err := makePassword(g, passwordBind)
		if err != nil {
			return
		}
	})
	symbolsButton.SetChecked(true)
	uppercaseButton := widget.NewCheck("A-Z", func(isChecked bool) {
		g.NoUpper = !isChecked
		err := makePassword(g, passwordBind)
		if err != nil {
			return
		}
	})
	uppercaseButton.SetChecked(true)

	optionsForm := widget.NewForm()
	optionsForm.Append(
		"Password",
		container.NewBorder(nil, nil, nil, refreshButton, passwordEntry),
	)

	optionsForm.Append(
		"Length",
		container.NewBorder(nil, nil, nil, lengthEntry, lengthSlider),
	)

	optionsForm.Append(
		"",
		container.NewGridWithColumns(3, digitsButton, symbolsButton, uppercaseButton),
	)

	return container.NewMax(optionsForm)
}

func makePassword(g *generatorOptions, externalString binding.ExternalString) error {
	if g.AllowSymbols {
		g.NumSymbols = g.Length * 25 / 100
	} else {
		g.NumSymbols = 0
	}
	if g.AllowDigits {
		g.NumDigits = g.Length * 25 / 100

	} else {
		g.NumDigits = 0
	}

	password, err := generator.Generate(g.Length, g.NumDigits, g.NumSymbols, g.NoUpper, g.AllowRepeat)
	if err != nil {
		return err
	}
	return externalString.Set(password)
}
