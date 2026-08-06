package models

import "time"

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
	ID                 uint64    `json:"id"`
	No                 *uint64   `json:"no"`
	Name               string    `json:"name"`
	IconURL            *string   `json:"iconUrl"`
	EvoChain           []string  `json:"evoChain"`
	BestPvpNatureIDs   []uint64  `json:"bestPvpNatureIds"`
	BestPvpNatureNames []string  `json:"bestPvpNatureNames,omitempty"`
	Notes              *string   `json:"notes"`
	EggGroupIDs        []uint64  `json:"eggGroupIds"`
	EggGroupNames      []string  `json:"eggGroupNames,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
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
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type MyPetInput struct {
	SpeciesID uint64         `json:"speciesId"`
	Gender    string         `json:"gender"`
	NatureID  uint64         `json:"natureId"`
	Medals    []PetMedalPick `json:"medals"`
}

type BreedingLineRun struct {
	ID        uint64    `json:"id"`
	LineID    uint64    `json:"lineId"`
	StudPetID uint64    `json:"studPetId"`
	DamPetID  uint64    `json:"damPetId"`
	Stud      *MyPet    `json:"stud,omitempty"`
	Dam       *MyPet    `json:"dam,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type BreedingLine struct {
	ID                 uint64              `json:"id"`
	Name               string              `json:"name"`
	TargetSpeciesID    uint64              `json:"targetSpeciesId"`
	TargetSpeciesName  string              `json:"targetSpeciesName,omitempty"`
	TargetSpeciesIcon  string              `json:"targetSpeciesIcon,omitempty"`
	ExpectedNatureID   uint64              `json:"expectedNatureId"`
	ExpectedNatureName string              `json:"expectedNatureName,omitempty"`
	Status             string              `json:"status"`
	StepsNote          *string             `json:"stepsNote"`
	Medals             []PetMedalPick      `json:"medals"`
	Schemes            []BreedingScheme    `json:"schemes"`
	Runs               []BreedingLineRun   `json:"runs"`
	MaxSlots           int                 `json:"maxSlots"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
}

type BreedingLineRunInput struct {
	StudPetID uint64 `json:"studPetId"`
	DamPetID  uint64 `json:"damPetId"`
}

type BreedingScheme struct {
	SchemeType int      `json:"schemeType"` // 1强 2弱公错 3弱母错
	Studs      []MyPet  `json:"studs"`
	Dams       []MyPet  `json:"dams"`
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
	WillRemove         bool   `json:"willRemove"` // 删除后产线无法成立
}

type PetDeleteImpact struct {
	Blocked      bool                   `json:"blocked"`
	Message      string                 `json:"message"`
	InProduction bool                   `json:"inProduction"`
	Lines        []PetDeleteLineImpact  `json:"lines"`
}
