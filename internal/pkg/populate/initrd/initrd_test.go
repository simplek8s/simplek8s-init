package initrd

// func Test_PopulateInitrdWithAssets(t *testing.T) {
// 	output := t.TempDir()

// 	user, err := user.Current()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	uid, err := strconv.Atoi(user.Uid)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	gid, err := strconv.Atoi(user.Gid)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if err := populateInitrdWithAssets(output, &copy.CopyOptions{Uid: uid, Gid: gid}); err != nil {
// 		t.Error(err)
// 	}
// 	dstAsset := filepath.Join(output, "usr/lib/systemd/system-preset/99-default.preset")
// 	if _, err := os.Stat(dstAsset); err != nil {
// 		t.Error(err)
// 	}
// }
