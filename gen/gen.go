package main

import (
	"github.com/lyricat/goutils/model/store"
	_ "github.com/lyricat/goutils/model/store/attachment"
)

func main() {
	store.Generate()
}
