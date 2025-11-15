package templates

import (
  "fmt"
)

func DebugPrint(path string) {
  data, err := content.ReadFile(path)
  if err != nil {
    fmt.Println("err:", err)
    return
  }
  fmt.Println("len:", len(data))
}
