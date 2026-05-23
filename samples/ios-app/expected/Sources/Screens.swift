// One SwiftUI view per declared screen.
// Replace the placeholder Text with your actual UI.
import SwiftUI

struct HomeView: View {
    var body: some View {
        NavigationStack {
            List {
                Section("Today's habits") {
                    Text("Daily checklist")
                    Text("Streak counter")
                    Text("Reset day")
                }
            }
            .navigationTitle("Today's habits")
        }
    }
}

struct HistoryView: View {
    var body: some View {
        NavigationStack {
            List {
                Section("Past 30 days") {
                    Text("Calendar view")
                    Text("Per-habit stats")
                }
            }
            .navigationTitle("Past 30 days")
        }
    }
}

