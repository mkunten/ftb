package main

type NdlOcrV3Line = NdlOcrV2Line
type NdlOcrV3Page = NdlOcrV2Page
type NdlOcrV3Book = NdlOcrV2Book
type NdlOcrV3ImgInfo = NdlOcrV2ImgInfo
type NdlOcrV3BookDetail = NdlOcrV2BookDetail

func NdlOcrV32BookText(ocrInfos []OCRInfo, isDetail bool) (*BookText, error) {
	return NdlOcrV22BookText(ocrInfos, isDetail)
}
