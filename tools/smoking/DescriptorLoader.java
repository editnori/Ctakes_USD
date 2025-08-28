package tools.smoking;

import java.net.URL;

import org.apache.uima.UIMAFramework;
import org.apache.uima.analysis_engine.AnalysisEngineDescription;
import org.apache.uima.resource.ResourceInitializationException;
import org.apache.uima.resource.ResourceSpecifier;
import org.apache.uima.util.InvalidXMLException;
import org.apache.uima.util.XMLInputSource;

final class DescriptorLoader {
    static AnalysisEngineDescription load(String resource) throws ResourceInitializationException {
        try {
            URL url = DescriptorLoader.class.getClassLoader().getResource(resource);
            if (url == null) {
                throw new ResourceInitializationException(new IllegalArgumentException("Resource not found: " + resource));
            }
            XMLInputSource xin = new XMLInputSource(url);
            ResourceSpecifier spec = UIMAFramework.getXMLParser().parseResourceSpecifier(xin);
            if (spec instanceof AnalysisEngineDescription) {
                return (AnalysisEngineDescription) spec;
            }
            throw new ResourceInitializationException(new InvalidXMLException(new Exception("Not an AE descriptor: " + resource)));
        } catch (Exception e) {
            if (e instanceof ResourceInitializationException) throw (ResourceInitializationException)e;
            throw new ResourceInitializationException(e);
        }
    }
    private DescriptorLoader() {}
}

