package main

import "myalgo/common"

func main() {
	// model.GetTrainData(1)
	common.GenerateUCRTestDataset("./dataset/UCRArchive_2018", "./dataset/ucrtest", 20, 100000)
}
