package constants

import "fmt"

const RednsSubsystemName = "redns"
const StraySubsystemName = "stray"

const (
	ActiveKeyTag   = "active"
	StagedKeyTag   = "staged"
	ArchivedKeyTag = "archived"

	MinTtlIfNotSpecified = 3600
)

func ScanAllActiveKeysTemplate() string {
	return fmt.Sprintf("%s/v1/records/*/"+ActiveKeyTag+"/*", RednsSubsystemName)
}

func ScanAllActiveStrayKeysTemplate() string {
	return fmt.Sprintf("%s/v1/records/*/%s/[^%s/]*", RednsSubsystemName, ActiveKeyTag, RednsSubsystemName)
}

func ScanSpecificActiveKeysTemplate(qname, qtype string) string {
	return fmt.Sprintf("%s/v1/records/%s/%s/%s/*", RednsSubsystemName, qname, qtype, ActiveKeyTag)
}

func GetGenerateKeyName(qname, qtype, generation string, recordId int) string {
	return fmt.Sprintf("%s/v1/records/%s/%s/"+StagedKeyTag+"/%s/%s/%d", RednsSubsystemName, qname, qtype, RednsSubsystemName, generation, recordId)
}

func ScanActiveKeysTemplate() string {
	return fmt.Sprintf("%s/v1/records/*/"+ActiveKeyTag+"/%s/*", RednsSubsystemName, RednsSubsystemName)
}

func ScanStagedKeysTemplate(generation string) string {
	return fmt.Sprintf("%s/v1/records/*/"+StagedKeyTag+"/%s/%s/*", RednsSubsystemName, RednsSubsystemName, generation)
}

func ScanGenerationKeysTemplate(role, generationId string) string {
	return fmt.Sprintf("%s/v1/records/*/"+role+"/%s/%s/*", RednsSubsystemName, RednsSubsystemName, generationId)
}

func ScanGenerationKeyTemplate(role, generationId, Id string) string {
	return fmt.Sprintf("%s/v1/records/*/*/"+role+"/%s/%s/%s", RednsSubsystemName, RednsSubsystemName, generationId, Id)
}

func ScanGenerationsKeysTemplate(role string) string {
	return fmt.Sprintf("%s/v1/records/*/"+role+"/%s/*", RednsSubsystemName, RednsSubsystemName)
}

func StrayEntityKeyTemplate(qname, qtype, recordId string) string {
	return fmt.Sprintf("%s/v1/records/%s/%s/"+ActiveKeyTag+"/%s/%s", RednsSubsystemName, qname, qtype, StraySubsystemName, recordId)
}
