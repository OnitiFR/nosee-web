package main

func OnitiMainEntry() {
	err := NotifySlack("Test notification Slack", true)
	if err != nil {
		panic(err)
	}
}
