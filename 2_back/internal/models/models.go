package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DBTime 兼容 SQLite TEXT 与 time.Time 扫描。
type DBTime time.Time

func (t *DBTime) Scan(src any) error {
	if src == nil {
		*t = DBTime{}
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		*t = DBTime(v)
		return nil
	case string:
		return t.parse(v)
	case []byte:
		return t.parse(string(v))
	default:
		return fmt.Errorf("cannot scan %T into DBTime", src)
	}
}

func (t *DBTime) parse(s string) error {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			*t = DBTime(parsed)
			return nil
		}
	}
	return fmt.Errorf("invalid time %q", s)
}

func (t DBTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}
	return time.Time(t).Format("2006-01-02 15:04:05"), nil
}

func (t DBTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t))
}

func (t *DBTime) UnmarshalJSON(b []byte) error {
	var tm time.Time
	if err := json.Unmarshal(b, &tm); err != nil {
		return err
	}
	*t = DBTime(tm)
	return nil
}

type EggGroup struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type Nature struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	Boost   string `json:"boost"`
	Penalty string `json:"penalty"`
}

type Medal struct {
	ID   uint64 `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type Species struct {
	ID                 uint64   `json:"id"`
	No                 *uint64  `json:"no"`
	Name               string   `json:"name"`
	IconURL            *string  `json:"iconUrl"`
	EvoChain           []string `json:"evoChain"`
	BestPvpNatureIDs   []uint64 `json:"bestPvpNatureIds"`
	BestPvpNatureNames []string `json:"bestPvpNatureNames,omitempty"`
	Notes              *string  `json:"notes"`
	EggGroupIDs        []uint64 `json:"eggGroupIds"`
	EggGroupNames      []string `json:"eggGroupNames,omitempty"`
	CreatedAt          DBTime   `json:"createdAt"`
	UpdatedAt          DBTime   `json:"updatedAt"`
}

type SpeciesInput struct {
	No               *uint64  `json:"no"`
	Name             string   `json:"name"`
	IconURL          *string  `json:"iconUrl"`
	EvoChain         []string `json:"evoChain"`
	BestPvpNatureIDs []uint64 `json:"bestPvpNatureIds"`
	Notes            *string  `json:"notes"`
	EggGroupIDs      []uint64 `json:"eggGroupIds"`
}

type PetMedalPick struct {
	MedalType string `json:"medalType"`
	MedalID   uint64 `json:"medalId"`
	MedalName string `json:"medalName,omitempty"`
}

type MyPet struct {
	ID          uint64         `json:"id"`
	SpeciesID   uint64         `json:"speciesId"`
	SpeciesName string         `json:"speciesName,omitempty"`
	Gender      string         `json:"gender"`
	NatureID    uint64         `json:"natureId"`
	NatureName  string         `json:"natureName,omitempty"`
	Status      string         `json:"status"`
	Medals      []PetMedalPick `json:"medals"`
	CreatedAt   DBTime         `json:"createdAt"`
	UpdatedAt   DBTime         `json:"updatedAt"`
}

type MyPetInput struct {
	SpeciesID uint64         `json:"speciesId"`
	Gender    string         `json:"gender"`
	NatureID  uint64         `json:"natureId"`
	Medals    []PetMedalPick `json:"medals"`
}

type BreedingLineRun struct {
	ID        uint64 `json:"id"`
	LineID    uint64 `json:"lineId"`
	StudPetID uint64 `json:"studPetId"`
	DamPetID  uint64 `json:"damPetId"`
	Stud      *MyPet `json:"stud,omitempty"`
	Dam       *MyPet `json:"dam,omitempty"`
	CreatedAt DBTime `json:"createdAt"`
}

type BreedingLine struct {
	ID                 uint64            `json:"id"`
	Name               string            `json:"name"`
	TargetSpeciesID    uint64            `json:"targetSpeciesId"`
	TargetSpeciesName  string            `json:"targetSpeciesName,omitempty"`
	TargetSpeciesIcon  string            `json:"targetSpeciesIcon,omitempty"`
	ExpectedNatureID   uint64            `json:"expectedNatureId"`
	ExpectedNatureName string            `json:"expectedNatureName,omitempty"`
	Status             string            `json:"status"`
	StepsNote          *string           `json:"stepsNote"`
	Medals             []PetMedalPick    `json:"medals"`
	Schemes            []BreedingScheme  `json:"schemes"`
	Runs               []BreedingLineRun `json:"runs"`
	MaxSlots           int               `json:"maxSlots"`
	CreatedAt          DBTime            `json:"createdAt"`
	UpdatedAt          DBTime            `json:"updatedAt"`
}

type BreedingLineRunInput struct {
	StudPetID uint64 `json:"studPetId"`
	DamPetID  uint64 `json:"damPetId"`
}

type BreedingScheme struct {
	SchemeType int     `json:"schemeType"`
	Studs      []MyPet `json:"studs"`
	Dams       []MyPet `json:"dams"`
}

type BreedingLineInput struct {
	Name             string         `json:"name"`
	TargetSpeciesID  uint64         `json:"targetSpeciesId"`
	ExpectedNatureID uint64         `json:"expectedNatureId"`
	Status           string         `json:"status"`
	StepsNote        *string        `json:"stepsNote"`
	Medals           []PetMedalPick `json:"medals"`
}

type BreedQueryResult struct {
	Species Species `json:"species"`
	Pets    []MyPet `json:"pets"`
}

type PetDeleteLineImpact struct {
	LineID             uint64 `json:"lineId"`
	LineName           string `json:"lineName"`
	TargetSpeciesName  string `json:"targetSpeciesName"`
	ExpectedNatureName string `json:"expectedNatureName"`
	WillRemove         bool   `json:"willRemove"`
}

type PetDeleteImpact struct {
	Blocked      bool                  `json:"blocked"`
	Message      string                `json:"message"`
	InProduction bool                  `json:"inProduction"`
	Lines        []PetDeleteLineImpact `json:"lines"`
}

type SpeciesDeleteImpact struct {
	Name      string   `json:"name"`
	PetCount  int      `json:"petCount"`
	LineCount int      `json:"lineCount"`
	LineNames []string `json:"lineNames"`
}
