package toolbox

import (
	"fmt"
	"strconv"
	"strings"
)

type TypeTag struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type StatRow struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value int    `json:"value"`
}

type TypeRelations struct {
	Counter     []TypeTag `json:"counter"`
	CounteredBy []TypeTag `json:"counteredBy"`
	Resist      []TypeTag `json:"resist"`
	ResistedBy  []TypeTag `json:"resistedBy"`
}

type SpeciesForm struct {
	Key           string        `json:"key"`
	Name          string        `json:"name"`
	SelectorLabel string        `json:"selectorLabel"`
	CatalogNo     *uint64       `json:"catalogNo,omitempty"`
	IconURL       string        `json:"iconUrl"`
	Types         []TypeTag     `json:"types"`
	Stage         int           `json:"stage"`
	StageLabel    string        `json:"stageLabel"`
	IsFinal       bool          `json:"isFinal"`
	IsLeader      bool          `json:"isLeader"`
	Stats         []StatRow     `json:"stats"`
	StatMax       int           `json:"statMax"`
	TypeRelations TypeRelations `json:"typeRelations"`
}

type SpeciesDetail struct {
	Name           string        `json:"name"`
	DefaultFormKey string        `json:"defaultFormKey"`
	Forms          []SpeciesForm `json:"forms"`
}

var defaultTypeLabels = map[string]string{
	"fire": "火", "water": "水", "grass": "草", "electric": "电",
	"ice": "冰", "ground": "地", "wing": "翼", "light": "光",
	"dark": "暗", "toxic": "毒", "insect": "虫", "dragon": "龙",
	"mechanic": "机械", "normal": "普通", "fight": "武", "psychic": "超能",
	"ghost": "幽", "rock": "岩", "steel": "钢", "fairy": "妖精",
	"moe": "萌", "evil": "恶",
	"demon": "恶", "phantom": "幻",
}

func typeTag(key string, learned map[string]string) TypeTag {
	key = strings.TrimSpace(key)
	if key == "" {
		return TypeTag{}
	}
	if lab := strings.TrimSpace(learned[key]); lab != "" {
		return TypeTag{Key: key, Label: lab}
	}
	if lab := defaultTypeLabels[key]; lab != "" {
		return TypeTag{Key: key, Label: lab}
	}
	return TypeTag{Key: key, Label: key}
}

func learnTypes(f catalogForm, learned map[string]string) {
	n := len(f.AttributeKeys)
	if len(f.AttributeLabels) < n {
		n = len(f.AttributeLabels)
	}
	for i := 0; i < n; i++ {
		k := strings.TrimSpace(f.AttributeKeys[i])
		lab := strings.TrimSpace(f.AttributeLabels[i])
		if k != "" && lab != "" {
			learned[k] = lab
		}
	}
}

func tagsOf(keys []string, learned map[string]string) []TypeTag {
	out := make([]TypeTag, 0, len(keys))
	seen := map[string]bool{}
	for _, k := range keys {
		t := typeTag(k, learned)
		if t.Key == "" || seen[t.Key] {
			continue
		}
		seen[t.Key] = true
		out = append(out, t)
	}
	return out
}

func formKey(f catalogForm) string {
	if f.BaseID > 0 {
		return strconv.Itoa(f.BaseID)
	}
	name := strings.TrimSpace(f.DisplayName)
	if name == "" {
		name = strings.TrimSpace(f.SelectorLabel)
	}
	return name
}

func toSpeciesForm(f catalogForm, catalogNo uint64, parentName string, isFinalItem bool, learned map[string]string) SpeciesForm {
	learnTypes(f, learned)
	types := make([]TypeTag, 0, len(f.AttributeKeys))
	for i, k := range f.AttributeKeys {
		lab := ""
		if i < len(f.AttributeLabels) {
			lab = f.AttributeLabels[i]
		}
		t := typeTag(k, learned)
		if lab != "" {
			t.Label = lab
		}
		if t.Key == "" {
			continue
		}
		types = append(types, t)
	}
	stats := make([]StatRow, 0, len(f.StatRows))
	max := f.StatChartMax
	for _, r := range f.StatRows {
		v := int(r.Value)
		if v > max {
			max = v
		}
		stats = append(stats, StatRow{Key: r.Key, Label: r.Label, Value: v})
	}
	if max <= 0 {
		max = 160
	}
	parent := strings.TrimSpace(parentName)
	name := strings.TrimSpace(f.DisplayName)
	if name == "" {
		name = parent
	}
	if lab := appearanceLabel(f); lab != "" && lab != name && name == parent {
		name = parent + "（" + lab + "）"
	}
	sel := name
	var no *uint64
	if catalogNo > 0 {
		n := catalogNo
		no = &n
	}
	leader := f.IsLeaderForm || strings.Contains(formBlob(f), "首领") || renamedDisplay(parent, f.DisplayName)
	return SpeciesForm{
		Key:           formKey(f),
		Name:          name,
		SelectorLabel: sel,
		CatalogNo:     no,
		IconURL:       liveIconURL(f.AvatarURL, f.BaseID),
		Types:         types,
		Stage:         f.Stage,
		StageLabel:    strings.TrimSpace(f.StageLabel),
		IsFinal:       isFinalItem && strings.TrimSpace(f.DisplayName) == parent && !leader,
		IsLeader:      leader,
		Stats:         stats,
		StatMax:       max,
		TypeRelations: TypeRelations{
			Counter:     tagsOf(f.TypeRelations.Counter, learned),
			CounteredBy: tagsOf(f.TypeRelations.CounteredBy, learned),
			Resist:      tagsOf(f.TypeRelations.Resist, learned),
			ResistedBy:  tagsOf(f.TypeRelations.ResistedBy, learned),
		},
	}
}

func wantedNames(name string, localChain []string, item catalogItem) map[string]bool {
	want := map[string]bool{strings.TrimSpace(name): true}
	for _, n := range localChain {
		n = strings.TrimSpace(n)
		if n != "" {
			want[n] = true
		}
	}
	form := defaultForm(item)
	for _, n := range evoChainOf(form, item.DisplayName) {
		want[n] = true
	}
	for _, f := range item.Forms {
		if n := strings.TrimSpace(f.DisplayName); n != "" {
			want[n] = true
		}
	}
	return want
}

func pickMatchedItem(query, formHint string, items []catalogItem) (catalogItem, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return catalogItem{}, false
	}
	base, hint := splitLocalName(query)
	if formHint == "" {
		formHint = hint
	}
	for _, it := range items {
		if strings.TrimSpace(it.DisplayName) == query {
			return it, true
		}
	}
	if base != "" && base != query {
		for _, it := range items {
			if strings.TrimSpace(it.DisplayName) != base {
				continue
			}
			if formHint == "" {
				return it, true
			}
			for _, f := range it.Forms {
				if appearanceMatches(f, formHint) || localSpeciesName(it, f) == query {
					return it, true
				}
			}
		}
	}
	for _, it := range items {
		for _, f := range it.Forms {
			if strings.TrimSpace(f.DisplayName) == query || localSpeciesName(it, f) == query {
				return it, true
			}
		}
	}
	return catalogItem{}, false
}

func appearanceMatches(f catalogForm, hint string) bool {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return false
	}
	return appearanceLabel(f) == hint || strings.TrimSpace(f.SelectorLabel) == hint
}

func itemHasAppearance(it catalogItem, hint string) bool {
	for _, f := range it.Forms {
		if appearanceMatches(f, hint) {
			return true
		}
	}
	return false
}

func isDefaultLineForm(it catalogItem, f catalogForm) bool {
	if skipPollutedForm(f) {
		return false
	}
	parent := strings.TrimSpace(it.DisplayName)
	disp := strings.TrimSpace(f.DisplayName)
	if disp != "" && parent != "" && disp != parent {
		return false
	}
	lab := appearanceLabel(f)
	return lab == "" || lab == "本来的样子" || f.IsDefaultForm
}

func formInLine(it catalogItem, f catalogForm, hint string) bool {
	if skipPollutedForm(f) {
		return false
	}
	if skipSyncForm(it, f) {
		return true
	}
	hint = strings.TrimSpace(hint)
	if hint != "" {
		if appearanceMatches(f, hint) {
			return true
		}
		return !itemHasAppearance(it, hint) && isDefaultLineForm(it, f)
	}
	return isDefaultLineForm(it, f)
}

func collectForms(matched catalogItem, related []catalogItem, want map[string]bool, learned map[string]string, formHint string) []SpeciesForm {
	type packed struct {
		form       catalogForm
		no         uint64
		parentName string
		isFinal    bool
	}
	var list []packed
	push := func(it catalogItem) {
		no := parseCatalogNo(it.CatalogNo)
		parent := strings.TrimSpace(it.DisplayName)
		for _, f := range it.Forms {
			if !formInLine(it, f, formHint) {
				continue
			}
			list = append(list, packed{form: f, no: no, parentName: parent, isFinal: it.IsFinalStage})
		}
	}
	push(matched)
	matchedName := strings.TrimSpace(matched.DisplayName)
	for _, it := range related {
		n := strings.TrimSpace(it.DisplayName)
		if n == matchedName {
			continue
		}
		keep := want[n]
		if !keep {
			for _, f := range it.Forms {
				if want[strings.TrimSpace(f.DisplayName)] {
					keep = true
					break
				}
			}
		}
		if keep {
			push(it)
		}
	}

	chain := evoChainOf(defaultForm(matched), matched.DisplayName)
	if inf := inferredChainFromItems(matched, related); len(inf) > len(chain) {
		chain = inf
	}
	chainIdx := map[string]int{}
	for i, n := range chain {
		if _, ok := chainIdx[n]; !ok {
			chainIdx[n] = i
		}
	}

	seen := map[string]bool{}
	out := make([]SpeciesForm, 0, len(list))
	// 先按进化链顺序放阶段体，再放首领化/地区形态
	addIfNew := func(p packed) {
		sf := toSpeciesForm(p.form, p.no, p.parentName, p.isFinal, learned)
		if sf.Key == "" || seen[sf.Key] {
			return
		}
		// 同一名字优先保留主条目默认形态
		seen[sf.Key] = true
		out = append(out, sf)
	}

	byName := map[string][]packed{}
	for _, p := range list {
		n := strings.TrimSpace(p.form.DisplayName)
		byName[n] = append(byName[n], p)
	}
	for _, n := range chain {
		for _, p := range byName[n] {
			addIfNew(p)
		}
	}
	for _, p := range list {
		n := strings.TrimSpace(p.form.DisplayName)
		if _, inChain := chainIdx[n]; inChain {
			continue
		}
		addIfNew(p)
	}
	disambiguateFormNames(out)
	return out
}

// disambiguateFormNames 工具箱常把同名形态挂多条（不同 id / 阶段 / 六维），下拉必须能分开。
func disambiguateFormNames(forms []SpeciesForm) {
	n := map[string]int{}
	for _, f := range forms {
		n[strings.TrimSpace(f.Name)]++
	}
	for i, f := range forms {
		name := strings.TrimSpace(f.Name)
		if name == "" || n[name] < 2 {
			continue
		}
		tag := strings.TrimSpace(f.StageLabel)
		if tag == "" && f.Stage > 0 {
			tag = strconv.Itoa(f.Stage) + "阶"
		}
		if f.IsLeader && !strings.Contains(name, "首领") {
			if tag != "" {
				tag += "·首领化"
			} else {
				tag = "首领化"
			}
		}
		if tag == "" {
			continue
		}
		forms[i].Name = name + "（" + tag + "）"
		forms[i].SelectorLabel = forms[i].Name
	}
}

func nameMatchesLocal(formName, localName string) bool {
	formName = strings.TrimSpace(formName)
	localName = strings.TrimSpace(localName)
	if formName == "" || localName == "" {
		return false
	}
	if formName == localName {
		return true
	}
	return strings.HasPrefix(formName, localName+"（") || strings.HasPrefix(formName, localName+"(")
}

func defaultFormKey(localName string, forms []SpeciesForm) string {
	localName = strings.TrimSpace(localName)
	_, hint := splitLocalName(localName)
	best, bestScore := "", -1
	for _, f := range forms {
		score := 0
		if f.Name == localName {
			score = 5
		} else if nameMatchesLocal(f.Name, localName) {
			score = 4
		} else if hint != "" && (f.SelectorLabel == hint || appearanceLabel(catalogForm{FormLabel: f.SelectorLabel}) == hint || strings.HasSuffix(f.Name, "（"+hint+"）")) {
			score = 3
		}
		if f.IsLeader {
			score--
		}
		if score > bestScore {
			bestScore = score
			best = f.Key
		}
	}
	if bestScore > 0 {
		return best
	}
	key, maxStage := "", -1
	for _, f := range forms {
		if f.IsLeader {
			continue
		}
		if f.Stage > maxStage {
			maxStage = f.Stage
			key = f.Key
		}
	}
	if key != "" {
		return key
	}
	if len(forms) > 0 {
		return forms[len(forms)-1].Key
	}
	return ""
}

func fetchMissingChain(chain []string, have map[string]bool) []catalogItem {
	var extra []catalogItem
	n := 0
	for _, name := range uniqueNames(chain) {
		if have[name] {
			continue
		}
		n++
		if n > 8 {
			break
		}
		pg, err := fetchCatalogPage(1, "", name)
		if err != nil {
			continue
		}
		it, ok := pickMatchedItem(name, "", pg.Results)
		if !ok {
			continue
		}
		extra = append(extra, it)
		have[name] = true
	}
	return extra
}

func fetchItemByCatalogNo(no uint64) (catalogItem, bool) {
	if no == 0 {
		return catalogItem{}, false
	}
	queries := []string{strconv.FormatUint(no, 10), fmt.Sprintf("No.%d", no)}
	for _, q := range queries {
		pg, err := fetchCatalogPage(1, "", q)
		if err != nil {
			continue
		}
		for _, it := range pg.Results {
			if parseCatalogNo(it.CatalogNo) == no {
				return it, true
			}
		}
	}
	return catalogItem{}, false
}

func inferPreEvoItems(it catalogItem) []catalogItem {
	if len(evoChainOf(defaultForm(it), it.DisplayName)) > 1 {
		return nil
	}
	no := parseCatalogNo(it.CatalogNo)
	if no == 0 {
		return nil
	}
	byNo := map[uint64]catalogItem{no: it}
	var extra []catalogItem
	for i := uint64(1); i <= 4; i++ {
		if no <= i {
			break
		}
		prevNo := no - i
		prev, ok := fetchItemByCatalogNo(prevNo)
		if !ok {
			break
		}
		byNo[prevNo] = prev
		if !isPreEvoItem(prev, byNo) {
			break
		}
		extra = append([]catalogItem{prev}, extra...)
	}
	return extra
}

// FetchSpeciesDetail 按名字实时拉形态 / 六维 / 克制，不写库。
func FetchSpeciesDetail(name string, localChain []string) (*SpeciesDetail, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("名称为空")
	}
	base, formHint := splitLocalName(name)
	queries := uniqueNames([]string{name, base, formHint})

	var matched catalogItem
	ok := false
	var related []catalogItem
	for _, q := range queries {
		pg, err := fetchCatalogPage(1, "", q)
		if err != nil {
			return nil, err
		}
		if it, hit := pickMatchedItem(name, formHint, pg.Results); hit {
			matched = it
			ok = true
			related = append(related, pg.Results...)
			break
		}
		if base != name {
			if it, hit := pickMatchedItem(base, formHint, pg.Results); hit {
				matched = it
				ok = true
				related = append(related, pg.Results...)
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("工具箱未找到「%s」", name)
	}

	want := wantedNames(name, localChain, matched)
	if base != "" {
		want[base] = true
	}
	have := map[string]bool{strings.TrimSpace(matched.DisplayName): true}
	for _, it := range related {
		have[strings.TrimSpace(it.DisplayName)] = true
	}
	for _, it := range inferPreEvoItems(matched) {
		n := strings.TrimSpace(it.DisplayName)
		if n == "" {
			continue
		}
		want[n] = true
		if !have[n] {
			related = append(related, it)
			have[n] = true
		}
	}
	related = append(related, fetchMissingChain(evoChainOf(defaultForm(matched), matched.DisplayName), have)...)
	for _, n := range localChain {
		n = strings.TrimSpace(n)
		b, _ := splitLocalName(n)
		if !have[n] {
			related = append(related, fetchMissingChain([]string{n}, have)...)
		}
		if b != n && !have[b] {
			related = append(related, fetchMissingChain([]string{b}, have)...)
		}
	}

	learned := map[string]string{}
	forms := collectForms(matched, related, want, learned, formHint)
	if len(forms) == 0 {
		return nil, fmt.Errorf("工具箱未返回形态数据")
	}
	return &SpeciesDetail{
		Name:           name,
		DefaultFormKey: defaultFormKey(name, forms),
		Forms:          forms,
	}, nil
}
