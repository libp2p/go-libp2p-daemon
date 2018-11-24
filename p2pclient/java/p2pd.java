
import java.util.Arrays;
public class p2pd {
    public static native void startDaemon();
    public static native void stopDaemon();
    static {
        try {
            
            System.loadLibrary ( "p2pd" ) ;
            
        } catch (UnsatisfiedLinkError e) {
          System.err.println("Native code library failed to load.\n" + e);
          System.exit(1);
        }
    }
    public static void main(String[] args) {
        startDaemon();
    }
}