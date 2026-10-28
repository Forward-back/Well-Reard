package types

import "github.com/yanyiwu/gojieba"

// Jieba is the process-wide Chinese tokenizer used by retrieval metrics and
// search helpers. gojieba allocates native dictionaries at construction.
var Jieba = gojieba.NewJieba()
