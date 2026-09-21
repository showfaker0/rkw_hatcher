package toolbox

import "testing"

func TestSplitLocalName(t *testing.T) {
	base, form := splitLocalName("鸭吉吉（燃了鸭）")
	if base != "鸭吉吉" || form != "燃了鸭" {
		t.Fatalf("got %q %q", base, form)
	}
	base, form = splitLocalName("魔力猫")
	if base != "魔力猫" || form != "" {
		t.Fatalf("got %q %q", base, form)
	}
}

func TestLocalSpeciesName(t *testing.T) {
	it := catalogItem{
		DisplayName: "鸭吉吉",
		Forms: []catalogForm{
			{DisplayName: "鸭吉吉", FormLabel: "蓬松的样子", SelectorLabel: "蓬松的样子", IsDefaultForm: true},
			{DisplayName: "鸭吉吉", FormLabel: "燃了鸭", SelectorLabel: "燃了鸭"},
		},
	}
	if g := localSpeciesName(it, it.Forms[1]); g != "鸭吉吉（燃了鸭）" {
		t.Fatalf("got %q", g)
	}

	moli := catalogItem{
		DisplayName: "魔力猫",
		Forms: []catalogForm{
			{DisplayName: "魔力猫", FormLabel: "默认形态", SelectorLabel: "魔力猫", IsDefaultForm: true},
			{DisplayName: "叶冕魔力猫", FormLabel: "默认形态", SelectorLabel: "叶冕魔力猫", IsLeaderForm: true},
			{DisplayName: "武斗酷猫", FormLabel: "默认形态", SelectorLabel: "武斗酷猫"},
		},
	}
	if g := localSpeciesName(moli, moli.Forms[0]); g != "魔力猫" {
		t.Fatalf("魔力猫 got %q", g)
	}
	if g := localSpeciesName(moli, moli.Forms[2]); g != "武斗酷猫" {
		t.Fatalf("武斗酷猫 got %q", g)
	}
}

func TestSkipAltForm(t *testing.T) {
	if !skipAltForm(catalogForm{IsLeaderForm: true, DisplayName: "叶冕魔力猫"}) {
		t.Fatal("首领化形态应排除")
	}
	if !skipAltForm(catalogForm{DisplayName: "被污染的布灵布灵", FormLabel: "被污染的布灵布灵"}) {
		t.Fatal("污染形态应排除")
	}
	if !skipAltForm(catalogForm{DisplayName: "嗜波螺", FormLabel: "被污染的样子"}) {
		t.Fatal("被污染的样子应排除")
	}
	if !skipAltForm(catalogForm{FormLabel: "突变形态"}) {
		t.Fatal("突变形态应排除")
	}
	if skipAltForm(catalogForm{DisplayName: "武斗酷猫", FormLabel: "默认形态"}) {
		t.Fatal("无首领标记时 skipAltForm 不单独拦换名")
	}
}

func TestSkipSyncForm(t *testing.T) {
	moli := catalogItem{
		DisplayName: "魔力猫",
		Forms: []catalogForm{
			{DisplayName: "魔力猫", FormLabel: "默认形态", IsDefaultForm: true},
			{DisplayName: "叶冕魔力猫", FormLabel: "默认形态", IsLeaderForm: true},
			{DisplayName: "武斗酷猫", FormLabel: "默认形态"},
		},
	}
	if skipSyncForm(moli, moli.Forms[0]) {
		t.Fatal("本尊应保留")
	}
	if !skipSyncForm(moli, moli.Forms[1]) {
		t.Fatal("标记首领化应排除")
	}
	if !skipSyncForm(moli, moli.Forms[2]) {
		t.Fatal("换名形态应排除")
	}

	dimo := catalogItem{
		DisplayName: "迪莫",
		Forms: []catalogForm{
			{DisplayName: "迪莫", FormLabel: "默认形态", IsDefaultForm: true},
			{DisplayName: "圣光迪莫", FormLabel: "默认形态"},
		},
	}
	if !skipSyncForm(dimo, dimo.Forms[1]) {
		t.Fatal("圣光迪莫应排除")
	}

	yaji := catalogItem{
		DisplayName: "鸭吉吉",
		Forms: []catalogForm{
			{DisplayName: "鸭吉吉", FormLabel: "蓬松的样子", IsDefaultForm: true},
			{DisplayName: "鸭吉吉", FormLabel: "燃了鸭"},
		},
	}
	if skipSyncForm(yaji, yaji.Forms[1]) {
		t.Fatal("同名样子应保留")
	}
}

func TestIsPreEvoItem(t *testing.T) {
	buling := catalogItem{
		CatalogNo: "No.464", DisplayName: "布灵", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 1, IsDefaultForm: true, DisplayName: "布灵"}},
	}
	buling2 := catalogItem{
		CatalogNo: "No.465", DisplayName: "布灵布灵", IsFinalStage: true,
		Forms: []catalogForm{
			{Stage: 2, IsDefaultForm: true, DisplayName: "布灵布灵"},
			{Stage: 2, DisplayName: "被污染的布灵布灵", FormLabel: "被污染的布灵布灵"},
		},
	}
	huadie := catalogItem{
		CatalogNo: "No.034", DisplayName: "化蝶", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 3, IsDefaultForm: true, DisplayName: "化蝶"}},
	}
	youying := catalogItem{
		CatalogNo: "No.035", DisplayName: "幽影树", IsFinalStage: true,
		Forms: []catalogForm{
			{Stage: 1, IsDefaultForm: true, DisplayName: "幽影树"},
			{Stage: 4, IsLeaderForm: true, DisplayName: "幻影荆棘"},
		},
	}
	yaji := catalogItem{
		CatalogNo: "No.011", DisplayName: "鸭吉吉", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 1, IsDefaultForm: true, DisplayName: "鸭吉吉"}},
	}
	byNo := map[uint64]catalogItem{
		464: buling, 465: buling2, 34: huadie, 35: youying, 11: yaji,
	}
	if !isPreEvoItem(buling, byNo) {
		t.Fatal("低阶形态应排除")
	}
	if isPreEvoItem(buling2, byNo) {
		t.Fatal("最终体应保留")
	}
	if isPreEvoItem(huadie, byNo) {
		t.Fatal("不相邻进化的最终体应保留")
	}
	if isPreEvoItem(yaji, byNo) {
		t.Fatal("独立 I 阶最终体应保留")
	}
}

func TestFormInLineAndDefaultKey(t *testing.T) {
	ling := catalogItem{
		DisplayName: "摇铃魔偶",
		Forms: []catalogForm{
			{DisplayName: "摇铃魔偶", IsDefaultForm: true, Stage: 2},
			{DisplayName: "被污染的摇铃魔偶", FormLabel: "被污染的摇铃魔偶", Stage: 2},
		},
	}
	if !formInLine(ling, ling.Forms[0], "") {
		t.Fatal("本尊应在切换列表")
	}
	if formInLine(ling, ling.Forms[1], "") {
		t.Fatal("污染形态不应在切换列表")
	}

	moli := catalogItem{
		DisplayName: "魔力猫",
		Forms: []catalogForm{
			{DisplayName: "魔力猫", FormLabel: "默认形态", IsDefaultForm: true},
			{DisplayName: "叶冕魔力猫", FormLabel: "默认形态", IsLeaderForm: true},
			{DisplayName: "武斗酷猫", FormLabel: "默认形态"},
		},
	}
	if !formInLine(moli, moli.Forms[2], "") {
		t.Fatal("换名首领化应在切换列表")
	}

	dimo := catalogItem{
		DisplayName: "迪莫",
		Forms: []catalogForm{
			{DisplayName: "迪莫", FormLabel: "默认形态", IsDefaultForm: true},
			{DisplayName: "圣光迪莫", FormLabel: "默认形态"},
			{DisplayName: "圣草迪莫", FormLabel: "默认形态"},
			{DisplayName: "圣火迪莫", FormLabel: "默认形态"},
			{DisplayName: "圣水迪莫", FormLabel: "默认形态"},
		},
	}
	for i := 1; i < len(dimo.Forms); i++ {
		if !formInLine(dimo, dimo.Forms[i], "") {
			t.Fatalf("%s 应在切换列表", dimo.Forms[i].DisplayName)
		}
	}

	shell := catalogItem{
		DisplayName: "水泡壳",
		Forms: []catalogForm{
			{DisplayName: "水泡壳", FormLabel: "本来的样子", IsDefaultForm: true, Stage: 3},
			{DisplayName: "水泡壳", FormLabel: "蜕皮时的样子", Stage: 3},
		},
	}
	if !formInLine(shell, shell.Forms[1], "蜕皮时的样子") {
		t.Fatal("所选样子应保留")
	}
	if formInLine(shell, shell.Forms[0], "蜕皮时的样子") {
		t.Fatal("其它样子不应混入")
	}

	heihua := catalogItem{
		DisplayName: "黑化加尔",
		Forms: []catalogForm{
			{DisplayName: "黑化加尔", FormLabel: "黑化的样子", IsDefaultForm: true, Stage: 3},
		},
	}
	jialing := catalogItem{
		DisplayName: "加灵",
		Forms: []catalogForm{
			{DisplayName: "加灵", FormLabel: "默认形态", IsDefaultForm: true, Stage: 1},
		},
	}
	if !formInLine(heihua, heihua.Forms[0], "黑化的样子") {
		t.Fatal("黑化样子应保留")
	}
	if !formInLine(jialing, jialing.Forms[0], "黑化的样子") {
		t.Fatal("无该样子的低阶应保留默认形态")
	}

	forms := []SpeciesForm{
		{Key: "3230", Name: "加灵", Stage: 1},
		{Key: "3231", Name: "加益", Stage: 2},
		{Key: "3232", Name: "加尔", Stage: 3},
		{Key: "3233", Name: "黑化加尔（黑化的样子）", Stage: 3},
	}
	if g := defaultFormKey("黑化加尔", forms); g != "3233" {
		t.Fatalf("黑化加尔 default %q", g)
	}
	if g := defaultFormKey("水泡壳（蜕皮时的样子）", []SpeciesForm{
		{Key: "3516", Name: "板板壳（蜕皮时的样子）", SelectorLabel: "板板壳（蜕皮时的样子）", Stage: 1},
		{Key: "3518", Name: "水泡壳（蜕皮时的样子）", SelectorLabel: "水泡壳（蜕皮时的样子）", Stage: 3},
	}); g != "3518" {
		t.Fatalf("蜕皮水泡壳 default %q", g)
	}
}

func TestDisambiguateFormNames(t *testing.T) {
	forms := []SpeciesForm{
		{Key: "3039", Name: "叮叮恶魔", Stage: 2, StageLabel: "II阶"},
		{Key: "5055", Name: "恶魔男爵", Stage: 4, StageLabel: "4阶", IsLeader: true},
		{Key: "13000172", Name: "恶魔男爵", Stage: 2, StageLabel: "II阶", IsLeader: true},
	}
	disambiguateFormNames(forms)
	if forms[0].Name != "叮叮恶魔" {
		t.Fatalf("本尊不应改名 %q", forms[0].Name)
	}
	if forms[1].Name != "恶魔男爵（4阶·首领化）" {
		t.Fatalf("4阶 %q", forms[1].Name)
	}
	if forms[2].Name != "恶魔男爵（II阶·首领化）" {
		t.Fatalf("II阶 %q", forms[2].Name)
	}
}

func TestInferredChain(t *testing.T) {
	buling := catalogItem{
		CatalogNo: "No.464", DisplayName: "布灵", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 1, IsDefaultForm: true, DisplayName: "布灵"}},
	}
	buling2 := catalogItem{
		CatalogNo: "No.465", DisplayName: "布灵布灵", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 2, IsDefaultForm: true, DisplayName: "布灵布灵"}},
	}
	huadie := catalogItem{
		CatalogNo: "No.034", DisplayName: "化蝶", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 3, IsDefaultForm: true, DisplayName: "化蝶"}},
	}
	youying := catalogItem{
		CatalogNo: "No.035", DisplayName: "幽影树", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 1, IsDefaultForm: true, DisplayName: "幽影树"}},
	}
	cefeng := catalogItem{
		CatalogNo: "No.458", DisplayName: "测风蝉", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 2, IsDefaultForm: true, DisplayName: "测风蝉"}},
	}
	liang := catalogItem{
		CatalogNo: "No.457", DisplayName: "量风碗", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 1, IsDefaultForm: true, DisplayName: "量风碗"}},
	}
	byNo := map[uint64]catalogItem{
		464: buling, 465: buling2, 34: huadie, 35: youying, 457: liang, 458: cefeng,
	}
	got := inferredChain(buling2, byNo)
	if len(got) != 2 || got[0] != "布灵" || got[1] != "布灵布灵" {
		t.Fatalf("布灵布灵 chain %v", got)
	}
	got = inferredChain(cefeng, byNo)
	if len(got) != 2 || got[0] != "量风碗" || got[1] != "测风蝉" {
		t.Fatalf("测风蝉 chain %v", got)
	}
	got = inferredChain(youying, byNo)
	if len(got) != 1 || got[0] != "幽影树" {
		t.Fatalf("幽影树不应接化蝶, got %v", got)
	}
	gui := catalogItem{
		CatalogNo: "No.455", DisplayName: "玳龟", IsFinalStage: false,
		Forms: []catalogForm{{Stage: 1, IsDefaultForm: true, DisplayName: "玳龟"}},
	}
	daita := catalogItem{
		CatalogNo: "No.456", DisplayName: "玳塔", IsFinalStage: true,
		Forms: []catalogForm{{Stage: 2, IsDefaultForm: true, DisplayName: "玳塔"}},
	}
	byNo[455] = gui
	byNo[456] = daita
	got = inferredChain(daita, byNo)
	if len(got) != 2 || got[0] != "玳龟" || got[1] != "玳塔" {
		t.Fatalf("玳塔 chain %v", got)
	}
	if !IsStubEvoChain("测风蝉", []string{"测风蝉"}) {
		t.Fatal("仅自身应视为占位链")
	}
	if IsStubEvoChain("测风蝉", []string{"量风碗", "测风蝉"}) {
		t.Fatal("完整链不是占位")
	}
	names := ChainNamesFromForms([]SpeciesForm{
		{Name: "量风碗", Stage: 1},
		{Name: "测风蝉", Stage: 2},
	}, "测风蝉")
	if len(names) != 2 || names[0] != "量风碗" || names[1] != "测风蝉" {
		t.Fatalf("forms chain %v", names)
	}
}
