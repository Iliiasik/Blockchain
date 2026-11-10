package resources

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
)

const english = `{
	"(Parent)": "(Parent)",
	"(Select one)": "(Select one)",
	"Advanced": "Advanced",
	"Cancel": "Cancel",
	"Confirm": "Confirm",
	"Copy": "Copy",
	"Create Folder": "Create Folder",
	"Cut": "Cut",
	"Enter filename": "Enter filename",
	"Error": "Error",
	"Favourites": "Favourites",
	"File": "File",
	"Folder": "Folder",
	"New Folder": "New Folder",
	"No": "No",
	"OK": "OK",
	"Open": "Open",
	"Paste": "Paste",
	"Quit": "Quit",
	"Redo": "Redo",
	"Save": "Save",
	"Select all": "Select all",
	"Show Hidden Files": "Show Hidden Files",
	"Undo": "Undo",
	"Yes": "Yes",
	"file.name": {"one": "Name", "few": "Name", "many": "Name", "other": "Name"},
	"file.parent": {"one": "Parent", "few": "Parent", "many": "Parent", "other": "Parent"},
	"example": "Example",
	"label.description": "Description",
	"friday": "Friday",
	"friday.short": "Fri",
	"monday": "Monday",
	"monday.short": "Mon",
	"saturday": "Saturday",
	"saturday.short": "Sat",
	"sunday": "Sunday",
	"sunday.short": "Sun",
	"thursday": "Thursday",
	"thursday.short": "Thu",
	"tuesday": "Tuesday",
	"tuesday.short": "Tue",
	"wednesday": "Wednesday",
	"wednesday.short": "Wed"
}`

func UseEnglish() {
	locale := lang.SystemLocale()
	if locale.LanguageString() == "en" {
		return
	}
	if err := lang.AddTranslationsForLocale([]byte(english), locale); err != nil {
		fyne.LogError("Could not force English interface strings", err)
	}
}
