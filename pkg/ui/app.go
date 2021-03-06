package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/devnull-twitch/godot-runner/pkg/storage"
	"github.com/sirupsen/logrus"
)

func Start() {
	app := app.New()
	win := app.NewWindow("Godot runner")
	win.Resize(fyne.Size{Width: 900, Height: 600})

	errLabel, errBind := errorTuple()

	globalForm, g := globalForm(errBind, win)
	envList := make([]*env, 0)
	envContainerBox := container.NewVBox()

	var oldWatcherCloser chan<- bool
	g.projectPathChangeHandler = func(s string) {
		if s != "" {
			if oldWatcherCloser != nil {
				oldWatcherCloser <- true
			}
			oldWatcherCloser = g.Watcher(s)
		}
	}
	g.projectFileChangeHandler = func() {
		for _, e := range envList {
			if e.isRunning && e.restartOnChange {
				e.restartNext = true
				e.currentProcess.Process.Kill()
			}
		}
	}

	win.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu(
			"File",
			fyne.NewMenuItem("Save config", func() {
				saveProject := &storage.Project{}
				saveProject.ExecPath, _ = g.exeBinding.Get()
				saveProject.ProjectPath, _ = g.projectPathBinding.Get()

				saveProject.Envs = make([]storage.Env, len(envList))
				for i, e := range envList {
					saveProject.Envs[i] = storage.Env{
						Name:            e.name,
						Scene:           e.scene,
						Arguments:       e.args,
						NoWindow:        e.noWindow,
						DebugCollisions: e.debugCollisions,
						DebugNavigation: e.debugNavigation,
					}
				}

				saveProject.Save()
			}),
			fyne.NewMenuItem("Load config", func() {
				loadProject := &storage.Project{}
				if err := loadProject.TryLoad(); err != nil {
					logrus.WithError(err).Warn("unable to load project")
					return
				}

				g.exeBinding.Set(loadProject.ExecPath)
				g.projectPathBinding.Set(loadProject.ProjectPath)

				for _, e := range loadProject.Envs {
					newEnv := createEnv(g)
					newEnv.name = e.Name
					newEnv.args = e.Arguments
					newEnv.scene = e.Scene
					newEnv.noWindow = e.NoWindow
					newEnv.debugCollisions = e.DebugCollisions
					newEnv.debugNavigation = e.DebugNavigation

					envList = append(envList, newEnv)

					newContainer, nameBinding := createListing(newEnv)
					envContainerBox.Add(newContainer)

					newEnv.AddListener(func() {
						nameBinding.Set(newEnv.name)
					})
				}
			}),
		),
	))

	windowContent := container.NewVBox(
		errLabel,
		globalForm,
		widget.NewButton("Add environment", func() {
			if path, _ := g.exeBinding.Get(); path == "" {
				return
			}

			newEnv := createEnv(g)
			formItems, newEnvBindings := newEnv.createEnvFormItems()
			formDia := dialog.NewForm("Create new environment", "Create", "Cancel", formItems, func(ok bool) {
				if ok {
					newEnv.name, _ = newEnvBindings.name.Get()
					newEnv.scene, _ = newEnvBindings.sceneBinding.Get()
					newEnv.args, _ = newEnvBindings.args.Get()

					envList = append(envList, newEnv)

					newContainer, nameBinding := createListing(newEnv)
					envContainerBox.Add(newContainer)

					newEnv.AddListener(func() {
						nameBinding.Set(newEnv.name)
					})
				}
			}, win)
			s := formDia.MinSize()
			if s.Width < win.Canvas().Size().Width*0.75 {
				s.Width = win.Canvas().Size().Width * 0.75
			}
			formDia.Resize(s)
			formDia.Show()
		}),
		envContainerBox,
	)
	win.SetContent(windowContent)
	win.ShowAndRun()
}
