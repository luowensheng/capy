// Root view with a TabView per declared screen.
import SwiftUI

struct RootView: View {
    var body: some View {
        TabView {
            HomeView()
                .tabItem {
                    Label("Today's habits", systemImage: "circle")
                }
            HistoryView()
                .tabItem {
                    Label("Past 30 days", systemImage: "circle")
                }
        }
    }
}
