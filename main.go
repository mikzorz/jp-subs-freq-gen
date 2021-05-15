package main

import (
  "fmt"
  "strings"

  "github.com/ikawaha/kagome-dict/ipa"
  "github.com/ikawaha/kagome/v2/tokenizer"
)

func main() {
  t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
  if err != nil {
    panic(err)
  }
  // wakati
  fmt.Println("---wakati---")
  seg := t.Wakati("お前はもう死んでいる")
  fmt.Println(seg)

  // tokenize
  fmt.Println("---tokenize---")
  tokens := t.Tokenize("お前はもう死んでいる")
  for _, token := range tokens {
    features := strings.Join(token.Features(), ",")
    fmt.Printf("%s\t%v\n", token.Surface, features)
  }
}
