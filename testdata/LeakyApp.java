import java.util.*;

/**
 * Simple app that creates a known heap pattern for testing hprof-analyzer.
 * Creates: HashMaps, ArrayLists, Strings, byte arrays, and a deliberate "leak".
 */
public class LeakyApp {
    // Deliberate "leak" — static list that keeps growing
    static final List<Map<String, byte[]>> leakyCache = new ArrayList<>();

    public static void main(String[] args) throws Exception {
        // Create some objects with known patterns
        for (int i = 0; i < 500; i++) {
            Map<String, byte[]> entry = new HashMap<>();
            entry.put("key-" + i, new byte[1024]); // 1KB per entry
            entry.put("data-" + i, new byte[4096]); // 4KB per entry
            leakyCache.add(entry);
        }

        // Create some standalone strings
        List<String> strings = new ArrayList<>();
        for (int i = 0; i < 200; i++) {
            strings.add("test-string-number-" + i + "-with-some-padding-to-make-it-longer");
        }

        // Create some nested structures
        Map<String, List<int[]>> nested = new HashMap<>();
        for (int i = 0; i < 50; i++) {
            List<int[]> list = new ArrayList<>();
            for (int j = 0; j < 10; j++) {
                list.add(new int[100]);
            }
            nested.put("group-" + i, list);
        }

        // Trigger heap dump
        // Using HotSpotDiagnosticMXBean to dump heap
        com.sun.management.HotSpotDiagnosticMXBean bean =
            java.lang.management.ManagementFactory.getPlatformMXBean(
                com.sun.management.HotSpotDiagnosticMXBean.class);

        String dumpPath = args.length > 0 ? args[0] : "testdata/leaky.hprof";
        bean.dumpHeap(dumpPath, true);
        System.out.println("Heap dump written to " + dumpPath);
    }
}
