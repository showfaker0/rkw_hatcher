package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func (a *API) exportSpecies(c *gin.Context) {
	rows, err := a.DB.Query(`
		SELECT s.id, s.`+"`no`"+`, s.name, s.evo_chain, s.notes,
		       (SELECT GROUP_CONCAT(n.name ORDER BY n.id SEPARATOR ',')
		        FROM species_best_natures sbn JOIN natures n ON n.id = sbn.nature_id
		        WHERE sbn.species_id = s.id),
		       (SELECT GROUP_CONCAT(e.name ORDER BY e.id SEPARATOR ',')
		        FROM species_egg_groups seg JOIN egg_groups e ON e.id = seg.egg_group_id
		        WHERE seg.species_id = s.id)
		FROM species s ORDER BY s.`+"`no`"+` IS NULL, s.`+"`no`"+`, s.id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheet := "图鉴"
	_ = f.SetSheetName("Sheet1", sheet)
	headers := []string{"id", "no", "name", "evo_chain", "best_pvp_natures", "egg_groups", "notes"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	r := 2
	for rows.Next() {
		var id uint64
		var no sql.NullInt64
		var name string
		var evo []byte
		var notes, natures, eggs interface{}
		if err := rows.Scan(&id, &no, &name, &evo, &notes, &natures, &eggs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		noVal := interface{}("")
		if no.Valid {
			noVal = no.Int64
		}
		vals := []interface{}{id, noVal, name, string(evo), fmt.Sprint(natures), fmt.Sprint(eggs), fmt.Sprint(notes)}
		if natures == nil {
			vals[4] = ""
		}
		if eggs == nil {
			vals[5] = ""
		}
		if notes == nil {
			vals[6] = ""
		}
		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, r)
			_ = f.SetCellValue(sheet, cell, v)
		}
		r++
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="species.xlsx"`)
	_ = f.Write(c.Writer)
}

func (a *API) importSpecies(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 file 字段"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	f, err := excelize.OpenReader(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表格为空"})
		return
	}
	header := map[string]int{}
	for i, h := range rows[0] {
		header[strings.TrimSpace(h)] = i
	}
	need := []string{"name", "evo_chain", "egg_groups"}
	for _, k := range need {
		if _, ok := header[k]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少列: " + k})
			return
		}
	}
	if _, ok := header["best_pvp_natures"]; !ok {
		if _, ok2 := header["best_pvp_nature"]; !ok2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少列: best_pvp_natures"})
			return
		}
	}
	okCount, fail := 0, []string{}
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		get := func(k string) string {
			idx, ok := header[k]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		name := get("name")
		if name == "" {
			continue
		}
		evoRaw := get("evo_chain")
		var chain []string
		if strings.HasPrefix(evoRaw, "[") {
			_ = json.Unmarshal([]byte(evoRaw), &chain)
		} else {
			for _, p := range strings.Split(evoRaw, "→") {
				p = strings.TrimSpace(p)
				if p != "" {
					chain = append(chain, p)
				}
			}
		}
		if len(chain) == 0 {
			chain = []string{name}
		}
		if chain[len(chain)-1] != name {
			chain = append(chain, name)
		}
		natureRaw := get("best_pvp_natures")
		if natureRaw == "" {
			natureRaw = get("best_pvp_nature")
		}
		var natureIDs []uint64
		for _, nn := range strings.Split(natureRaw, ",") {
			nn = strings.TrimSpace(nn)
			if nn == "" {
				continue
			}
			var nid uint64
			if err := a.DB.QueryRow(`SELECT id FROM natures WHERE name=?`, nn).Scan(&nid); err != nil {
				fail = append(fail, fmt.Sprintf("行%d 性格不存在:%s", i+1, nn))
				continue
			}
			natureIDs = append(natureIDs, nid)
		}
		if len(natureIDs) == 0 {
			fail = append(fail, fmt.Sprintf("行%d 无有效最佳性格", i+1))
			continue
		}
		eggNames := strings.Split(get("egg_groups"), ",")
		var eggIDs []uint64
		for _, en := range eggNames {
			en = strings.TrimSpace(en)
			if en == "" {
				continue
			}
			var eid uint64
			if err := a.DB.QueryRow(`SELECT id FROM egg_groups WHERE name=?`, en).Scan(&eid); err != nil {
				fail = append(fail, fmt.Sprintf("行%d 蛋组不存在:%s", i+1, en))
				continue
			}
			eggIDs = append(eggIDs, eid)
		}
		if len(eggIDs) == 0 {
			fail = append(fail, fmt.Sprintf("行%d 无有效蛋组", i+1))
			continue
		}
		notes := get("notes")
		var notesPtr *string
		if notes != "" {
			notesPtr = &notes
		}
		var noPtr *uint64
		if noRaw := get("no"); noRaw != "" {
			if n, err := strconv.ParseUint(noRaw, 10, 64); err == nil && n > 0 {
				noPtr = &n
			}
		}
		evo, _ := json.Marshal(chain)
		tx, err := a.DB.Begin()
		if err != nil {
			fail = append(fail, err.Error())
			continue
		}
		var sid uint64
		err = tx.QueryRow(`SELECT id FROM species WHERE name=?`, name).Scan(&sid)
		if err == nil {
			_, err = tx.Exec(`UPDATE species SET `+"`no`"+`=?, evo_chain=?, notes=? WHERE id=?`, nullUint64(noPtr), evo, nullStr(notesPtr), sid)
		} else {
			res, e2 := tx.Exec(`INSERT INTO species (`+"`no`"+`, name, evo_chain, notes) VALUES (?,?,?,?)`, nullUint64(noPtr), name, evo, nullStr(notesPtr))
			err = e2
			if err == nil {
				id64, _ := res.LastInsertId()
				sid = uint64(id64)
			}
		}
		if err != nil {
			_ = tx.Rollback()
			fail = append(fail, fmt.Sprintf("行%d %v", i+1, err))
			continue
		}
		if err := a.replaceSpeciesEggGroups(tx, sid, eggIDs); err != nil {
			_ = tx.Rollback()
			fail = append(fail, fmt.Sprintf("行%d %v", i+1, err))
			continue
		}
		if err := a.replaceSpeciesBestNatures(tx, sid, natureIDs); err != nil {
			_ = tx.Rollback()
			fail = append(fail, fmt.Sprintf("行%d %v", i+1, err))
			continue
		}
		if err := tx.Commit(); err != nil {
			fail = append(fail, err.Error())
			continue
		}
		okCount++
	}
	c.JSON(http.StatusOK, gin.H{"imported": okCount, "failed": fail})
}

func (a *API) exportPets(c *gin.Context) {
	rows, err := a.DB.Query(`
		SELECT p.id, s.name, p.gender, n.name
		FROM my_pets p
		JOIN species s ON s.id = p.species_id
		JOIN natures n ON n.id = p.nature_id
		ORDER BY p.id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	f := excelize.NewFile()
	sheet := "我的宠物"
	_ = f.SetSheetName("Sheet1", sheet)
	headers := []string{"id", "species", "gender", "nature", "medals"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	r := 2
	for rows.Next() {
		var id uint64
		var species, gender, nature string
		if err := rows.Scan(&id, &species, &gender, &nature); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		medals, _ := a.loadPetMedals(id)
		parts := make([]string, 0, len(medals))
		for _, m := range medals {
			parts = append(parts, m.MedalType+":"+m.MedalName)
		}
		vals := []interface{}{id, species, gender, nature, strings.Join(parts, ",")}
		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, r)
			_ = f.SetCellValue(sheet, cell, v)
		}
		r++
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="pets.xlsx"`)
	_ = f.Write(c.Writer)
}

func (a *API) importPets(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 file 字段"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	f, err := excelize.OpenReader(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "表格为空"})
		return
	}
	header := map[string]int{}
	for i, h := range rows[0] {
		header[strings.TrimSpace(h)] = i
	}
	for _, k := range []string{"species", "gender", "nature"} {
		if _, ok := header[k]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少列: " + k})
			return
		}
	}
	okCount, fail := 0, []string{}
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		get := func(k string) string {
			idx, ok := header[k]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		speciesName := get("species")
		if speciesName == "" {
			continue
		}
		var speciesID, natureID uint64
		if err := a.DB.QueryRow(`SELECT id FROM species WHERE name=?`, speciesName).Scan(&speciesID); err != nil {
			fail = append(fail, fmt.Sprintf("行%d 图鉴不存在", i+1))
			continue
		}
		if err := a.DB.QueryRow(`SELECT id FROM natures WHERE name=?`, get("nature")).Scan(&natureID); err != nil {
			fail = append(fail, fmt.Sprintf("行%d 性格不存在", i+1))
			continue
		}
		gender := get("gender")
		if gender != "公" && gender != "母" {
			fail = append(fail, fmt.Sprintf("行%d 性别无效", i+1))
			continue
		}
		tx, err := a.DB.Begin()
		if err != nil {
			fail = append(fail, err.Error())
			continue
		}
		res, err := tx.Exec(`INSERT INTO my_pets (species_id, gender, nature_id) VALUES (?,?,?)`,
			speciesID, gender, natureID)
		if err != nil {
			_ = tx.Rollback()
			fail = append(fail, fmt.Sprintf("行%d %v", i+1, err))
			continue
		}
		id64, _ := res.LastInsertId()
		medalRaw := get("medals")
		if medalRaw != "" {
			for _, part := range strings.Split(medalRaw, ",") {
				part = strings.TrimSpace(part)
				kv := strings.SplitN(part, ":", 2)
				if len(kv) != 2 {
					continue
				}
				typ, name := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
				var mid uint64
				if err := tx.QueryRow(`SELECT id FROM medals WHERE type=? AND name=?`, typ, name).Scan(&mid); err != nil {
					continue
				}
				_, _ = tx.Exec(`INSERT INTO my_pet_medals (my_pet_id, medal_type, medal_id) VALUES (?,?,?)`, id64, typ, mid)
			}
		}
		if err := tx.Commit(); err != nil {
			fail = append(fail, err.Error())
			continue
		}
		okCount++
		_ = strconv.FormatUint(uint64(id64), 10)
	}
	c.JSON(http.StatusOK, gin.H{"imported": okCount, "failed": fail})
}
