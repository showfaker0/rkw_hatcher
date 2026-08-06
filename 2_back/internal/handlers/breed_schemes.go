package handlers

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"rkw_hatcher/internal/models"
)

// buildBreedSchemes 产线推荐核心（与前端规则一致）
// excludeLineID 保留供调用方传编辑中的产线 ID；excludePetID>0 时排除该宠（用于删宠预览）
func (a *API) buildBreedSchemes(targetSpeciesID, natureID uint64, medals []models.PetMedalPick, excludeLineID, excludePetID uint64) ([]models.BreedingScheme, error) {
	_ = excludeLineID
	targetEggs, _, err := a.loadSpeciesEggGroups(targetSpeciesID)
	if err != nil {
		return nil, err
	}
	if len(targetEggs) == 0 {
		return nil, nil
	}
	eggSet := map[uint64]struct{}{}
	for _, id := range targetEggs {
		eggSet[id] = struct{}{}
	}

	need := map[string]uint64{}
	for _, m := range medals {
		if m.MedalID == 0 || strings.TrimSpace(m.MedalType) == "" {
			continue
		}
		need[m.MedalType] = m.MedalID
	}

	rows, err := a.DB.Query(`
		SELECT p.id, p.species_id, s.name, p.gender, p.nature_id, n.name, p.status, p.created_at, p.updated_at
		FROM my_pets p
		JOIN species s ON s.id = p.species_id
		JOIN natures n ON n.id = p.nature_id
		ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studOK, studBad, damOK, damBad []models.MyPet
	for rows.Next() {
		var p models.MyPet
		if err := rows.Scan(&p.ID, &p.SpeciesID, &p.SpeciesName, &p.Gender, &p.NatureID, &p.NatureName, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if excludePetID > 0 && p.ID == excludePetID {
			continue
		}
		// 推荐与忙碌/空闲无关
		p.Medals, _ = a.loadPetMedals(p.ID)
		if !petHasAllMedals(p.Medals, need) {
			continue
		}
		natureOK := p.NatureID == natureID
		if p.Gender == "母" {
			if p.SpeciesID != targetSpeciesID {
				continue
			}
			if natureOK {
				damOK = append(damOK, p)
			} else {
				damBad = append(damBad, p)
			}
			continue
		}
		if p.Gender != "公" {
			continue
		}
		eggs, _, _ := a.loadSpeciesEggGroups(p.SpeciesID)
		if !eggIntersect(eggs, eggSet) {
			continue
		}
		if natureOK {
			studOK = append(studOK, p)
		} else {
			studBad = append(studBad, p)
		}
	}

	var out []models.BreedingScheme
	if len(studOK) > 0 && len(damOK) > 0 {
		out = append(out, models.BreedingScheme{SchemeType: 1, Studs: studOK, Dams: damOK})
		return out, nil
	}
	if len(studBad) > 0 && len(damOK) > 0 {
		out = append(out, models.BreedingScheme{SchemeType: 2, Studs: studBad, Dams: damOK})
	}
	if len(studOK) > 0 && len(damBad) > 0 {
		out = append(out, models.BreedingScheme{SchemeType: 3, Studs: studOK, Dams: damBad})
	}
	if len(out) == 0 && len(studBad) > 0 && len(damBad) > 0 {
		out = append(out, models.BreedingScheme{SchemeType: 4, Studs: studBad, Dams: damBad})
	}
	return out, nil
}

func petHasAllMedals(have []models.PetMedalPick, need map[string]uint64) bool {
	if len(need) == 0 {
		return true
	}
	got := map[string]uint64{}
	for _, m := range have {
		got[m.MedalType] = m.MedalID
	}
	for typ, id := range need {
		if got[typ] != id {
			return false
		}
	}
	return true
}

func eggIntersect(eggs []uint64, set map[uint64]struct{}) bool {
	for _, id := range eggs {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}

func medalsFingerprint(medals []models.PetMedalPick) string {
	type pair struct {
		t  string
		id uint64
	}
	ps := make([]pair, 0, len(medals))
	for _, m := range medals {
		if m.MedalID == 0 || strings.TrimSpace(m.MedalType) == "" {
			continue
		}
		ps = append(ps, pair{m.MedalType, m.MedalID})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].t == ps[j].t {
			return ps[i].id < ps[j].id
		}
		return ps[i].t < ps[j].t
	})
	parts := make([]string, 0, len(ps))
	for _, p := range ps {
		parts = append(parts, fmt.Sprintf("%s:%d", p.t, p.id))
	}
	return strings.Join(parts, "|")
}

func (a *API) lineMedalsConflict(targetSpeciesID, natureID uint64, medals []models.PetMedalPick, excludeLineID uint64) (bool, error) {
	want := medalsFingerprint(medals)
	rows, err := a.DB.Query(`
		SELECT b.id FROM breeding_lines b
		WHERE b.target_species_id=? AND b.expected_nature_id=?`, targetSpeciesID, natureID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return false, err
		}
		if excludeLineID > 0 && id == excludeLineID {
			continue
		}
		ms, err := a.loadLineMedals(id)
		if err != nil {
			return false, err
		}
		if medalsFingerprint(ms) == want {
			return true, nil
		}
	}
	return false, nil
}

func (a *API) loadLineMedals(lineID uint64) ([]models.PetMedalPick, error) {
	rows, err := a.DB.Query(`
		SELECT blm.medal_type, blm.medal_id, m.name
		FROM breeding_line_medals blm
		JOIN medals m ON m.id = blm.medal_id
		WHERE blm.line_id=? ORDER BY blm.medal_type`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.PetMedalPick
	for rows.Next() {
		var p models.PetMedalPick
		if err := rows.Scan(&p.MedalType, &p.MedalID, &p.MedalName); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if list == nil {
		list = []models.PetMedalPick{}
	}
	return list, nil
}

func (a *API) loadLineSchemes(lineID uint64) ([]models.BreedingScheme, error) {
	rows, err := a.DB.Query(`SELECT scheme_type FROM breeding_line_schemes WHERE line_id=? ORDER BY scheme_type`, lineID)
	if err != nil {
		return nil, err
	}
	var types []int
	for rows.Next() {
		var t int
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return nil, err
		}
		types = append(types, t)
	}
	rows.Close()

	var out []models.BreedingScheme
	for _, t := range types {
		sch := models.BreedingScheme{SchemeType: t, Studs: []models.MyPet{}, Dams: []models.MyPet{}}
		prows, err := a.DB.Query(`
			SELECT p.id, p.species_id, s.name, p.gender, p.nature_id, n.name, p.status, p.created_at, p.updated_at, blsp.role
			FROM breeding_line_scheme_pets blsp
			JOIN my_pets p ON p.id = blsp.pet_id
			JOIN species s ON s.id = p.species_id
			JOIN natures n ON n.id = p.nature_id
			WHERE blsp.line_id=? AND blsp.scheme_type=?
			ORDER BY blsp.role, p.id`, lineID, t)
		if err != nil {
			return nil, err
		}
		for prows.Next() {
			var p models.MyPet
			var role string
			if err := prows.Scan(&p.ID, &p.SpeciesID, &p.SpeciesName, &p.Gender, &p.NatureID, &p.NatureName, &p.Status, &p.CreatedAt, &p.UpdatedAt, &role); err != nil {
				prows.Close()
				return nil, err
			}
			p.Medals, _ = a.loadPetMedals(p.ID)
			if role == "stud" {
				sch.Studs = append(sch.Studs, p)
			} else {
				sch.Dams = append(sch.Dams, p)
			}
		}
		prows.Close()
		out = append(out, sch)
	}
	if out == nil {
		out = []models.BreedingScheme{}
	}
	return out, nil
}

func (a *API) replaceLineMedals(tx *sql.Tx, lineID uint64, medals []models.PetMedalPick) error {
	if _, err := tx.Exec(`DELETE FROM breeding_line_medals WHERE line_id=?`, lineID); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, m := range medals {
		if m.MedalID == 0 || strings.TrimSpace(m.MedalType) == "" {
			continue
		}
		if _, ok := seen[m.MedalType]; ok {
			continue
		}
		seen[m.MedalType] = struct{}{}
		var typ string
		if err := tx.QueryRow(`SELECT type FROM medals WHERE id=?`, m.MedalID).Scan(&typ); err != nil {
			return err
		}
		if typ != m.MedalType {
			return simpleError("奖章与类型不匹配")
		}
		if _, err := tx.Exec(`INSERT INTO breeding_line_medals (line_id, medal_type, medal_id) VALUES (?,?,?)`,
			lineID, m.MedalType, m.MedalID); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) replaceLineSchemes(tx *sql.Tx, lineID uint64, schemes []models.BreedingScheme) error {
	if _, err := tx.Exec(`DELETE FROM breeding_line_scheme_pets WHERE line_id=?`, lineID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM breeding_line_schemes WHERE line_id=?`, lineID); err != nil {
		return err
	}
	for _, sch := range schemes {
		if _, err := tx.Exec(`INSERT INTO breeding_line_schemes (line_id, scheme_type) VALUES (?,?)`, lineID, sch.SchemeType); err != nil {
			return err
		}
		for _, p := range sch.Studs {
			if _, err := tx.Exec(`INSERT INTO breeding_line_scheme_pets (line_id, scheme_type, pet_id, role) VALUES (?,?,?,'stud')`,
				lineID, sch.SchemeType, p.ID); err != nil {
				return err
			}
		}
		for _, p := range sch.Dams {
			if _, err := tx.Exec(`INSERT INTO breeding_line_scheme_pets (line_id, scheme_type, pet_id, role) VALUES (?,?,?,'dam')`,
				lineID, sch.SchemeType, p.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectSchemePetIDs(schemes []models.BreedingScheme) []uint64 {
	seen := map[uint64]struct{}{}
	var ids []uint64
	for _, sch := range schemes {
		for _, p := range sch.Studs {
			if _, ok := seen[p.ID]; !ok {
				seen[p.ID] = struct{}{}
				ids = append(ids, p.ID)
			}
		}
		for _, p := range sch.Dams {
			if _, ok := seen[p.ID]; !ok {
				seen[p.ID] = struct{}{}
				ids = append(ids, p.ID)
			}
		}
	}
	return ids
}

func (a *API) setPetsStatus(tx *sql.Tx, ids []uint64, status string) error {
	for _, id := range ids {
		if _, err := tx.Exec(`UPDATE my_pets SET status=? WHERE id=?`, status, id); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) releasePetsIfNotInRuns(tx *sql.Tx, ids []uint64) error {
	for _, id := range ids {
		var n int
		err := tx.QueryRow(`
			SELECT COUNT(*) FROM breeding_line_runs
			WHERE stud_pet_id=? OR dam_pet_id=?`, id, id).Scan(&n)
		if err != nil {
			return err
		}
		if n == 0 {
			if _, err := tx.Exec(`UPDATE my_pets SET status='空闲中' WHERE id=?`, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *API) lineSchemePetIDs(tx *sql.Tx, lineID uint64) ([]uint64, error) {
	rows, err := tx.Query(`SELECT DISTINCT pet_id FROM breeding_line_scheme_pets WHERE line_id=?`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func calcLineMaxSlots(schemes []models.BreedingScheme) int {
	studs := map[uint64]struct{}{}
	dams := map[uint64]struct{}{}
	for _, sch := range schemes {
		for _, p := range sch.Studs {
			studs[p.ID] = struct{}{}
		}
		for _, p := range sch.Dams {
			dams[p.ID] = struct{}{}
		}
	}
	n := len(studs)
	if len(dams) < n {
		n = len(dams)
	}
	if n > 5 {
		n = 5
	}
	return n
}

func (a *API) loadLineRuns(lineID uint64) ([]models.BreedingLineRun, error) {
	rows, err := a.DB.Query(`
		SELECT id, line_id, stud_pet_id, dam_pet_id, created_at
		FROM breeding_line_runs WHERE line_id=? ORDER BY id`, lineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.BreedingLineRun
	for rows.Next() {
		var r models.BreedingLineRun
		if err := rows.Scan(&r.ID, &r.LineID, &r.StudPetID, &r.DamPetID, &r.CreatedAt); err != nil {
			return nil, err
		}
		if stud, err := a.fetchPetBrief(r.StudPetID); err == nil {
			r.Stud = &stud
		}
		if dam, err := a.fetchPetBrief(r.DamPetID); err == nil {
			r.Dam = &dam
		}
		list = append(list, r)
	}
	if list == nil {
		list = []models.BreedingLineRun{}
	}
	return list, nil
}

func (a *API) fetchPetBrief(id uint64) (models.MyPet, error) {
	var p models.MyPet
	err := a.DB.QueryRow(`
		SELECT p.id, p.species_id, s.name, p.gender, p.nature_id, n.name, p.status, p.created_at, p.updated_at
		FROM my_pets p
		JOIN species s ON s.id = p.species_id
		JOIN natures n ON n.id = p.nature_id
		WHERE p.id=?`, id).
		Scan(&p.ID, &p.SpeciesID, &p.SpeciesName, &p.Gender, &p.NatureID, &p.NatureName, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	p.Medals, _ = a.loadPetMedals(p.ID)
	return p, nil
}

func (a *API) countGlobalRuns(tx *sql.Tx) (int, error) {
	var n int
	var err error
	if tx != nil {
		err = tx.QueryRow(`SELECT COUNT(*) FROM breeding_line_runs`).Scan(&n)
	} else {
		err = a.DB.QueryRow(`SELECT COUNT(*) FROM breeding_line_runs`).Scan(&n)
	}
	return n, err
}

func (a *API) countLineRuns(tx *sql.Tx, lineID uint64) (int, error) {
	var n int
	err := tx.QueryRow(`SELECT COUNT(*) FROM breeding_line_runs WHERE line_id=?`, lineID).Scan(&n)
	return n, err
}

func (a *API) petInLineScheme(tx *sql.Tx, lineID, petID uint64, role string) (bool, error) {
	var n int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM breeding_line_scheme_pets
		WHERE line_id=? AND pet_id=? AND role=?`, lineID, petID, role).Scan(&n)
	return n > 0, err
}

func (a *API) dropInvalidLineRuns(tx *sql.Tx, lineID uint64) error {
	rows, err := tx.Query(`SELECT id, stud_pet_id, dam_pet_id FROM breeding_line_runs WHERE line_id=?`, lineID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type runRow struct {
		id, stud, dam uint64
	}
	var bad []runRow
	for rows.Next() {
		var r runRow
		if err := rows.Scan(&r.id, &r.stud, &r.dam); err != nil {
			return err
		}
		okStud, err := a.petInLineScheme(tx, lineID, r.stud, "stud")
		if err != nil {
			return err
		}
		okDam, err := a.petInLineScheme(tx, lineID, r.dam, "dam")
		if err != nil {
			return err
		}
		if !okStud || !okDam {
			bad = append(bad, r)
		}
	}
	for _, r := range bad {
		if _, err := tx.Exec(`DELETE FROM breeding_line_runs WHERE id=?`, r.id); err != nil {
			return err
		}
		if err := a.releasePetsIfNotInRuns(tx, []uint64{r.stud, r.dam}); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) fetchBreedingLine(id uint64) (models.BreedingLine, error) {
	var b models.BreedingLine
	var note sql.NullString
	var icon sql.NullString
	err := a.DB.QueryRow(`
		SELECT b.id, b.name, b.target_species_id, s.name, s.icon_url, b.expected_nature_id, n.name,
		       b.status, b.steps_note, b.created_at, b.updated_at
		FROM breeding_lines b
		JOIN species s ON s.id = b.target_species_id
		JOIN natures n ON n.id = b.expected_nature_id
		WHERE b.id=?`, id).
		Scan(&b.ID, &b.Name, &b.TargetSpeciesID, &b.TargetSpeciesName, &icon,
			&b.ExpectedNatureID, &b.ExpectedNatureName, &b.Status, &note, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return b, err
	}
	if icon.Valid {
		b.TargetSpeciesIcon = icon.String
	}
	b.StepsNote = scanNullString(note)
	b.Medals, err = a.loadLineMedals(b.ID)
	if err != nil {
		return b, err
	}
	b.Schemes, err = a.loadLineSchemes(b.ID)
	if err != nil {
		return b, err
	}
	b.MaxSlots = calcLineMaxSlots(b.Schemes)
	b.Runs, err = a.loadLineRuns(b.ID)
	return b, err
}
