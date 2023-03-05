package word

import "strings"

func ToCamel(word string) string {
	// 将字符串分割为单词，并使用 Title 函数将每个单词的首字母大写
	words := strings.Split(word, "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}
	// 将单词连接在一起，形成一个驼峰字符串
	camelCase := strings.Join(words, "")
	return camelCase
}
