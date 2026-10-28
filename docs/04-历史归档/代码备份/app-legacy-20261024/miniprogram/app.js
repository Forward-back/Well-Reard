App({
  onLaunch() {
    const settings = wx.getStorageSync("gaze_settings");
    if (!settings) {
      wx.setStorageSync("gaze_settings", {
        baseUrl: "http://localhost:8080",
        apiKey: "",
        selectedKnowledgeBaseId: "",
        locale: "zh"
      });
    }
  }
});
