package main

import (
	"github.com/Sirupsen/logrus"
)

func logImageEvent(imageID, refName, action string) {
	logrus.Debugf("{action: %q, image: %q, ref: %q}", action, imageID, refName)
}
