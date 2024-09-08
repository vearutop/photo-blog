package cloudflare

// {"@cf/llava-hf/llava-1.5-7b-hf":{"description":" The image features a beautiful view of a body of water, with a large
// building situated next to it. The building has a unique design, as it is made of bricks and has a blue and black
// color scheme. The building also has a clock on its side, adding to its charm. The scene is further enhanced by the
// presence of a clock tower, which stands tall and proudly overlooking the water. The combination of the building, the
// water, and the clock tower creates a picturesque and memorable scene.","elapsedTimeMs":7012},
// "@cf/facebook/detr-resnet-50":[{"score":0.00038208975456655025,"label":"clock","box":{"xmin":992,"ymin":24,"xmax":
// 1073,"ymax":146}},{"score":0.00033619155874475837,"label":"clock","box":{"xmin":984,"ymin":12,"xmax":1066,"ymax":146}},
// {"score":0.00030715574393980205,"label":"clock","box":{"xmin":1001,"ymin":12,"xmax":1087,"ymax":147}},{"score":0.00026796586462296546,"label":"clock","box":{"xmin":978,"ymin":14,"xmax":1063,"ymax":148}},{"score":0.00022972737497184426,"label":"clock","box":{"xmin":868,"ymin":4,"xmax":952,"ymax":157}},{"score":0.00022324558813124895,"label":"clock","box":{"xmin":977,"ymin":12,"xmax":1067,"ymax":149}},{"score":0.0001523054961580783,"label":"clock","box":{"xmin":953,"ymin":2,"xmax":1131,"ymax":170}},{"score":0.0001288951898459345,"label":"person","box":{"xmin":1128,"ymin":1180,"xmax":1181,"ymax":1317}},{"score":0.00010501006909180433,"label":"clock","box":{"xmin":451,"ymin":65534,"xmax":842,"ymax":116}},{"score":0.00009954364941222593,"label":"clock","box":{"xmin":906,"ymin":12,"xmax":1008,"ymax":147}},{"score":0.00008585948671679944,"label":"person","box":{"xmin":1133,"ymin":1198,"xmax":1191,"ymax":1695}},{"score":0.00006678515637759119,"label":"train","box":{"xmin":220,"ymin":1756,"xmax":841,"ymax":1800}},{"score":0.000052219413191778585,"label":"bench","box":{"xmin":346,"ymin":1743,"xmax":974,"ymax":1800}},{"score":0.00005126635005581193,"label":"airplane","box":{"xmin":491,"ymin":1148,"xmax":535,"ymax":1174}},{"score":0.000050112670578528196,"label":"toilet","box":{"xmin":845,"ymin":1489,"xmax":927,"ymax":1584}},{"score":0.00004695616371463984,"label":"train","box":{"xmin":283,"ymin":1712,"xmax":838,"ymax":1799}},{"score":0.00004303732202970423,"label":"train","box":{"xmin":183,"ymin":1741,"xmax":1018,"ymax":1800}},{"score":0.00004204976721666753,"label":"airplane","box":{"xmin":512,"ymin":1131,"xmax":624,"ymax":1174}},{"score":0.000039667949749855325,"label":"train","box":{"xmin":229,"ymin":1700,"xmax":1035,"ymax":1800}},{"score":0.000039174294215627015,"label":"train","box":{"xmin":534,"ymin":1174,"xmax":616,"ymax":1202}},{"score":0.00003738763552973978,"label":"bench","box":{"xmin":402,"ymin":1740,"xmax":842,"ymax":1800}},{"score":0.00003713914702530019,"label":"clock","box":{"xmin":67,"ymin":0,"xmax":293,"ymax":85}},{"score":0.00003379146437509917,"label":"bench","box":{"xmin":141,"ymin":1749,"xmax":840,"ymax":1801}},{"score":0.00003351770283188671,"label":"toilet","box":{"xmin":896,"ymin":1519,"xmax":1084,"ymax":1689}},{"score":0.00002932960524049122,"label":"airplane","box":{"xmin":521,"ymin":1155,"xmax":588,"ymax":1181}},{"score":0.000028938218747498468,"label":"bench","box":{"xmin":957,"ymin":1693,"xmax":1065,"ymax":1800}},{"score":0.000028919683245476335,"label":"bench","box":{"xmin":923,"ymin":1711,"xmax":1071,"ymax":1800}},{"score":0.000028117407055106014,"label":"person","box":{"xmin":475,"ymin":1127,"xmax":500,"ymax":1158}},{"score":0.000027755088012781925,"label":"bench","box":{"xmin":419,"ymin":1727,"xmax":947,"ymax":1801}},{"score":0.00002723586658248678,"label":"person","box":{"xmin":798,"ymin":1494,"xmax":892,"ymax":1590}},{"score":0.000027142592443851754,"label":"person","box":{"xmin":479,"ymin":1129,"xmax":504,"ymax":1160}},{"score":0.00002646935718075838,"label":"toilet","box":{"xmin":920,"ymin":1588,"xmax":1061,"ymax":1712}},{"score":0.000026149991754209623,"label":"boat","box":{"xmin":525,"ymin":1153,"xmax":591,"ymax":1179}},{"score":0.000025806182748056017,"label":"person","box":{"xmin":479,"ymin":1137,"xmax":506,"ymax":1166}},{"score":0.000025231844119844027,"label":"fire hydrant","box":{"xmin":930,"ymin":1673,"xmax":1090,"ymax":1800}},{"score":0.00002516635686333757,"label":"person","box":{"xmin":476,"ymin":1129,"xmax":500,"ymax":1159}},{"score":0.000024074735847534612,"label":"person","box":{"xmin":465,"ymin":1129,"xmax":492,"ymax":1161}},{"score":0.00002346075234527234,"label":"train","box":{"xmin":547,"ymin":1157,"xmax":618,"ymax":1182}},{"score":0.000023063823391566984,"label":"bench","box":{"xmin":52,"ymin":1751,"xmax":735,"ymax":1801}},{"score":0.0000216297667066101,"label":"bench","box":{"xmin":974,"ymin":1686,"xmax":1077,"ymax":1800}},{"score":0.000021550622477661818,"label":"fire hydrant","box":{"xmin":998,"ymin":1695,"xmax":1094,"ymax":1800}},{"score":0.000021085281332489103,"label":"boat","box":{"xmin":629,"ymin":1151,"xmax":686,"ymax":1183}},{"score":0.000021057800040580332,"label":"boat","box":{"xmin":646,"ymin":1158,"xmax":697,"ymax":1189}},{"score":0.00001960210283868946,"label":"bench","box":{"xmin":494,"ymin":1730,"xmax":891,"ymax":1801}},{"score":0.0000192712959687924,"label":"fire hydrant","box":{"xmin":966,"ymin":1664,"xmax":1072,"ymax":1800}},{"score":0.000019039614926441573,"label":"train","box":{"xmin":570,"ymin":1160,"xmax":646,"ymax":1188}},{"score":0.000018767599613056518,"label":"bench","box":{"xmin":988,"ymin":1694,"xmax":1071,"ymax":1800}},{"score":0.000018110646124114282,"label":"boat","box":{"xmin":565,"ymin":1156,"xmax":641,"ymax":1184}},{"score":0.000018055870896205306,"label":"fire hydrant","box":{"xmin":998,"ymin":1693,"xmax":1089,"ymax":1799}},{"score":0.00001804741441446822,"label":"clock","box":{"xmin":231,"ymin":187,"xmax":782,"ymax":283}},{"score":0.000017664806364336982,"label":"train","box":{"xmin":216,"ymin":1714,"xmax":980,"ymax":1800}},{"score":0.00001757445170369465,"label":"fire hydrant","box":{"xmin":986,"ymin":1676,"xmax":1098,"ymax":1800}},{"score":0.000017036147255566902,"label":"boat","box":{"xmin":594,"ymin":1154,"xmax":664,"ymax":1184}},{"score":0.000016826934370328672,"label":"bench","box":{"xmin":982,"ymin":1685,"xmax":1070,"ymax":1800}},{"score":0.000016717942344257608,"label":"bench","box":{"xmin":632,"ymin":1737,"xmax":951,"ymax":1800}},{"score":0.000016183626939891838,"label":"train","box":{"xmin":214,"ymin":1740,"xmax":903,"ymax":1800}},{"score":0.000016115547623485327,"label":"train","box":{"xmin":338,"ymin":1599,"xmax":951,"ymax":1746}},{"score":0.000015868612535996363,"label":"person","box":{"xmin":2,"ymin":1733,"xmax":262,"ymax":1800}},{"score":0.00001576973227201961,"label":"fire hydrant","box":{"xmin":993,"ymin":1661,"xmax":1089,"ymax":1799}},{"score":0.000015360430552391335,"label":"clock","box":{"xmin":279,"ymin":65535,"xmax":1009,"ymax":154}},{"score":0.000014535839909513015,"label":"fire hydrant","box":{"xmin":967,"ymin":1563,"xmax":1119,"ymax":1799}},{"score":0.000012884294847026467,"label":"fire hydrant","box":{"xmin":1003,"ymin":1521,"xmax":1147,"ymax":1743}},{"score":0.000012303456060180906,"label":"toilet","box":{"xmin":912,"ymin":1322,"xmax":1133,"ymax":1673}},{"score":0.00001204544969368726,"label":"train","box":{"xmin":44,"ymin":1735,"xmax":753,"ymax":1800}},{"score":0.000011447240467532538,"label":"bench","box":{"xmin":184,"ymin":1767,"xmax":599,"ymax":1800}},{"score":0.000010534670764172915,"label":"train","box":{"xmin":286,"ymin":1749,"xmax":904,"ymax":1800}},{"score":0.000010110490620718338,"label":"clock","box":{"xmin":1,"ymin":112,"xmax":171,"ymax":272}},{"score":0.000009763774869497865,"label":"toilet","box":{"xmin":893,"ymin":1208,"xmax":1146,"ymax":1763}},{"score":0.000008682805855642073,"label":"person","box":{"xmin":0,"ymin":1641,"xmax":178,"ymax":1790}},{"score":0.000008666928806633223,"label":"train","box":{"xmin":43,"ymin":347,"xmax":1070,"ymax":1805}},{"score":0.000008433936272922438,"label":"train","box":{"xmin":65525,"ymin":645,"xmax":1009,"ymax":1816}},{"score":0.00000782904498919379,"label":"train","box":{"xmin":700,"ymin":1734,"xmax":1006,"ymax":1800}},{"score":0.000007455652394128265,"label":"train","box":{"xmin":64,"ymin":1748,"xmax":807,"ymax":1800}},{"score":0.00000741817075322615,"label":"person","box":{"xmin":0,"ymin":1737,"xmax":198,"ymax":1800}},{"score":0.0000072153625296778046,"label":"train","box":{"xmin":112,"ymin":1710,"xmax":1011,"ymax":1800}},{"score":0.0000070676669565727934,"label":"train","box":{"xmin":119,"ymin":1669,"xmax":959,"ymax":1800}},{"score":0.000006072041287552565,"label":"train","box":{"xmin":50,"ymin":722,"xmax":1062,"ymax":1454}},{"score":0.000005996908384986455,"label":"bench","box":{"xmin":0,"ymin":1690,"xmax":306,"ymax":1800}},{"score":0.000004783302756550256,"label":"person","box":{"xmin":23,"ymin":998,"xmax":210,"ymax":1152}},{"score":0.0000044594312385015655,"label":"train","box":{"xmin":26,"ymin":248,"xmax":660,"ymax":1665}},{"score":0.00000439634595750249,"label":"person","box":{"xmin":1,"ymin":1114,"xmax":138,"ymax":1237}},{"score":0.00000394562766814488,"label":"airplane","box":{"xmin":463,"ymin":731,"xmax":703,"ymax":1176}},{"score":0.000003805320829997072,"label":"train","box":{"xmin":62,"ymin":1700,"xmax":731,"ymax":1800}},{"score":0.000003650997541626566,"label":"person","box":{"xmin":582,"ymin":67,"xmax":918,"ymax":1535}},{"score":0.000003162254870403558,"label":"clock","box":{"xmin":859,"ymin":65508,"xmax":1173,"ymax":1261}},{"score":0.0000031313909403252183,"label":"toilet","box":{"xmin":879,"ymin":397,"xmax":1155,"ymax":1690}},{"score":0.000002994840542669408,"label":"train","box":{"xmin":105,"ymin":1615,"xmax":912,"ymax":1784}},{"score":0.0000028525660127343144,"label":"train","box":{"xmin":55,"ymin":215,"xmax":967,"ymax":1671}},{"score":0.0000027092144136986462,"label":"person","box":{"xmin":1,"ymin":927,"xmax":117,"ymax":1225}},{"score":0.0000023055345081957057,"label":"clock","box":{"xmin":460,"ymin":917,"xmax":854,"ymax":1538}},{"score":0.0000021920156996202422,"label":"toilet","box":{"xmin":526,"ymin":144,"xmax":1048,"ymax":1686}},{"score":0.0000021497121451830026,"label":"toilet","box":{"xmin":909,"ymin":236,"xmax":1157,"ymax":1690}},{"score":0.000002038123056991026,"label":"train","box":{"xmin":65513,"ymin":201,"xmax":679,"ymax":1762}},{"score":0.0000014939772654543049,"label":"train","box":{"xmin":5,"ymin":296,"xmax":441,"ymax":1711}},{"score":0.0000014683650988445152,"label":"train","box":{"xmin":0,"ymin":342,"xmax":250,"ymax":1679}},{"score":0.0000013765853736913414,"label":"train","box":{"xmin":8,"ymin":248,"xmax":463,"ymax":1684}},{"score":0.0000012440060572771472,"label":"refrigerator","box":{"xmin":0,"ymin":153,"xmax":161,"ymax":1664}},{"score":0.000001234609271705267,"label":"person","box":{"xmin":1,"ymin":280,"xmax":170,"ymax":1106}},{"score":8.732899914321024e-7,"label":"train","box":{"xmin":0,"ymin":199,"xmax":207,"ymax":1683}},{"score":7.587495360894536e-7,"label":"person","box":{"xmin":0,"ymin":188,"xmax":150,"ymax":1688}}],"@cf/meta/m2m100-1.2b":{"russian":{"translated_text":"На изображении изображен красивый вид на тело воды, с большим зданием, расположенным рядом с ним. Здание имеет уникальный дизайн, так как оно сделано из кирпича и имеет синюю и черную цветовую схему. Здание также имеет часы на своей стороне, добавляя к своему очарованию. Сцена еще больше усиливается наличием часовой башни, которая стоит высоко и с гордостью смотрит на воду. Сочетание здания, воды и часовой башни создает живописную и запоминающуюся сцену.","elapsedTimeMs":10686},"german":{"translated_text":"Das Bild zeigt einen schönen Blick auf einen Wasserkörper, mit einem großen Gebäude neben ihm. Das Gebäude hat ein einzigartiges Design, da es aus Ziegeln besteht und ein blaues und schwarzes Farbschema hat. Das Gebäude hat auch eine Uhr auf seiner Seite, was seinen Charme hinzufügt. Die Szene wird durch das Vorhandensein eines Uhrturms, der hoch steht und stolz auf das Wasser blickt, weiter verstärkt. Die Kombination des Gebäudes, des Wassers und des Uhrturms schafft eine malerische und unvergessliche Szene.","elapsedTimeMs":6446}}}
type WorkerResp struct {
	LlavaDesc struct {
		Description   string `json:"description"`
		ElapsedTimeMs int    `json:"elapsedTimeMs"`
	} `json:"@cf/llava-hf/llava-1.5-7b-hf"`
	Detr []struct {
		Score float64 `json:"score"`
		Label string  `json:"label"`
		Box   struct {
			Xmin int `json:"xmin"`
			Ymin int `json:"ymin"`
			Xmax int `json:"xmax"`
			Ymax int `json:"ymax"`
		} `json:"box"`
	} `json:"@cf/facebook/detr-resnet-50"`
	Translate map[string]struct {
		TranslatedText string `json:"translated_text"`
		ElapsedTimeMs  int    `json:"elapsedTimeMs"`
	} `json:"@cf/meta/m2m100-1.2b"`
}

type DetrBox struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
	Box   struct {
		Xmin int `json:"xmin"`
		Ymin int `json:"ymin"`
		Xmax int `json:"xmax"`
		Ymax int `json:"ymax"`
	}
}

type Worker struct {
	URL string
}

type WorkerReq struct {
	Image     []byte `json:"image"`
	Llama     bool   `json:"llama"`
	Detr      bool   `json:"detr"`
	Translate []string
}

//func (w *Worker) ProcessImage(ctx context.Context, req WorkerReq) {
//	buf := bytes.NewBuffer(nil)
//	image := multipart.NewWriter(buf)
//
//	//image.
//
//	r, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, image)
//	if err != nil {
//		return err
//	}
//
//	if req.Llama {
//		r.Header.Set("X-Llama", "1")
//	}
//
//	if req.Detr {
//		r.Header.Set("X-Detr", "1")
//	}
//
//}
