package main

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var files []string
var splittedFiles []string
var mainFileName string

func main() {
	myApp := app.New()

	myWindow := myApp.NewWindow("Postman-Preparer")
	myWindow.Resize(fyne.NewSize(800, 600))

	osTempDir := os.TempDir()

	// Main menu
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Quit", func() {
			cleanUpTempFiles(osTempDir)
			myApp.Quit()
		}),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			dialog.ShowCustom("About", "Close", container.NewVBox(
				widget.NewLabel("Welcome to the Postman-Preparer-App, a simple Desktop app created in Go with Fyne."),
				widget.NewLabel("Version: v0.1"),
				widget.NewLabel("Author: Matthäus Malek"),
			), myWindow)
		}),
		fyne.NewMenuItem("Help", func() {
			dialog.ShowCustom("Help", "Close", container.NewVBox(
				widget.NewLabel("This App is supposed to help you with preparing the csv data for the postman-request"),
				widget.NewLabel("You can upload your csv file. After uploading your file, you can split it in smaller chunks"),
				widget.NewLabel("Because the API Gateway, only supports 29 sec of response time, we need to send an amount of max 25K emails"),
				widget.NewLabel("After the split, hit the 'makePostmanReady' Button"),
				widget.NewLabel("It will ask you where to save your file, which will held the prepared content."),
				widget.NewLabel("Copy-Paste your content to postman and hit the SEND button. All good :)"),
			), myWindow)
		}),
	)
	mainMenu := fyne.NewMainMenu(
		fileMenu,
		helpMenu,
	)
	myWindow.SetMainMenu(mainMenu)

	// Define a welcome text centered
	text := canvas.NewText("Welcome", color.White)
	text.Alignment = fyne.TextAlignCenter

	// result labels
	uploadLabel := widget.NewLabel("result uploaded file:")
	splitLabel := widget.NewLabel("you have splited files:")

	// canvas line
	line := canvas.NewLine(color.White)
	line.StrokeWidth = 5

	// Define Upload button
	uploadCsvBtn := widget.NewButton("Upload CSV", func() {
		files, err := UploadCsvBtn(myWindow, uploadLabel)
		if err != nil {
			fmt.Println("could not upload File", err)
			return
		}
		fmt.Println("files on upload", files)
		// uploadLabel.SetText("Upload Successful File Name" + mainFileName)
	})
	uploadCsvBtn.Importance = widget.HighImportance

	// Define Split Button
	splitFilesBtn := widget.NewButton("Split Files", func() {
		splittedFiles, err := splitCsvFiles(files[0], splitLabel)
		if err != nil {
			fmt.Println("where not able to split files")
			return
		}
		fmt.Println(splittedFiles)
		fmt.Printf("Uploaded Files has been split into %d files", len(splittedFiles))

	})
	splitFilesBtn.Importance = widget.DangerImportance

	// this makePostmanBodyBtn prepares all known files in tmp folder
	// for the request in postman
	makePostmanBodyBtn := widget.NewButton("makePostmanBody", func() {
		for i, file := range splittedFiles {
			ok := testFileSize(file)
			if ok {
				err := convertCsvFileToPostmanReadyBody(file, int64(i), myWindow)
				if err != nil {
					fmt.Println("could prepare csv file for postman", err)
				}
			}
		}
	})
	makePostmanBodyBtn.Importance = widget.MediumImportance

	// Display a vertical box containing text, image and button
	box := container.NewVBox(
		text,
		uploadCsvBtn,
		uploadLabel,
		splitFilesBtn,
		splitLabel,
		line,
		makePostmanBodyBtn,
	)
	// Display our content
	myWindow.SetContent(box)
	// Close the App when Escape key is pressed
	myWindow.Canvas().SetOnTypedKey(func(keyEvent *fyne.KeyEvent) {
		if keyEvent.Name == fyne.KeyEscape {
			myApp.Quit()
		}
	})
	myWindow.ShowAndRun()
}

func UploadCsvBtn(myWindow fyne.Window, uploadLabel *widget.Label) ([]string, error) {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		defer reader.Close()

		tempDir := os.TempDir()
		tempFile, err := ioutil.TempFile(tempDir, "*.csv")
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		defer tempFile.Close()
		files = append(files, tempFile.Name())
		// Copy the contents of the selected file to the temp file
		_, err = io.Copy(tempFile, reader)
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		fmt.Println("FileName is:", files)
		mainFileName = path.Base(reader.URI().String())
		uploadLabel.SetText("Upload Successful File Name " + mainFileName)
	}, myWindow)
	return files, nil
}

func convertCsvFileToPostmanReadyBody(filePath string, fileNumber int64, myWindow fyne.Window) error {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the CSV file
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	// Process the data
	var emailList []string
	for _, record := range records {
		email := record[0]
		emailList = append(emailList, email)
	}
	emailString := strings.Join(emailList, ",")

	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		defer writer.Close()

		// Write the processed data to the chosen file
		_, err = io.WriteString(writer, emailString)
		if err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		fmt.Println("Processed data saved to: ", writer.URI().Name())

	}, myWindow)

	return nil
}

func splitCsvFiles(filePath string, splitLabel *widget.Label) ([]string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("could not open Csv File. SplitButton here")
		return nil, err
	}
	defer file.Close()

	// Read the CSV file
	reader := csv.NewReader(file)

	// counter to keep track of how many rows have been processed
	counter := 0

	// open first file
	tempFile, err := os.Create(filePath + "_part1.csv")
	if err != nil {
		return nil, err
	}
	defer tempFile.Close()
	splittedFiles = append(splittedFiles, tempFile.Name())
	fmt.Println("First File has been created")

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		counter++

		// create new file when counter reaches 25000
		if counter%25000 == 0 {
			tempFile.Close()
			fmt.Println("New File will be created. 25000 entries has been red")
			tempFile, err = os.Create(filePath + "_part" + strconv.Itoa(counter/25000+1) + ".csv")
			if err != nil {
				return nil, err
			}
			splittedFiles = append(splittedFiles, tempFile.Name())
		}
		writer := csv.NewWriter(tempFile)
		writer.Write(record)
		writer.Flush()
	}
	splitLabel.SetText("You have split your primary file into " + fmt.Sprint(len(splittedFiles)) + " files")

	return splittedFiles, nil
}

func testFileSize(filePath string) bool {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Could not open file:", err)
		return false
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Count the number of rows
	rowCount := 0
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error reading file:", err)
			return false
		}
		rowCount++
	}

	// Compare the number of rows to 25K
	if rowCount > 25000 {
		fmt.Println("File has more than 25K emails.")
		return false
	} else {
		fmt.Println("File has 25K or less emails.")
		return true
	}
}

func cleanUpTempFiles(tempDir string) {
	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		fmt.Println("could not delete all files from tempDir")
	}

	for _, file := range files {
		os.Remove(filepath.Join(tempDir, file.Name()))
	}
}
