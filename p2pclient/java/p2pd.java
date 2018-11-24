
public class p2pd {
    public static native void startD();
    public static native void stopD();
    static {
        try {
            
            System.loadLibrary ( "p2pd" ) ;
            
        } catch (UnsatisfiedLinkError e) {
          System.err.println("Native code library failed to load.\n" + e);
          System.exit(1);
        }
    }
    public static void main(String[] args) {
        startD();
    }
}