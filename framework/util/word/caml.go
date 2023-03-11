package word

import "strings"

// ToTitleCamel 将下划线分割的字符串转换为驼峰字符串
// class_id => ClassId
func ToTitleCamel(word string) string {
	// 将字符串分割为单词，并使用 Title 函数将每个单词的首字母大写
	words := strings.Split(word, "_")
	for i, word := range words {
		words[i] = strings.Title(word)
	}
	// 将单词连接在一起，形成一个驼峰字符串
	camelCase := strings.Join(words, "")
	return camelCase
}

// ToNormalCamel 将下划线分割的字符串转换为驼峰字符串
// class_id => classId
func ToNormalCamel(word string) string {
	// 将字符串分割为单词，并使用 Title 函数将每个单词的首字母大写
	words := strings.Split(word, "_")
	for i, word := range words {
		if i == 0 {
			words[i] = strings.ToLower(word)
			continue
		}
		words[i] = strings.Title(word)
	}
	// 将单词连接在一起，形成一个驼峰字符串
	camelCase := strings.Join(words, "")
	return camelCase
}
