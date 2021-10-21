# epub-comic-repacker
extract the image files from epub comic books, and repack them into zip files, fits for any epub files, expecially downloaded from vol.me or mox.moe

### build
after "go build", the porgamme will run in CMD like mode without gui, but you can add an icon to make it look better, just follow this up:

#### in Windows
Create file epub-comic-repacker.manifest
```
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
    <assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="epub-comic-repacker" type="win32"/>
    <dependency>
        <dependentAssembly>
            <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
        </dependentAssembly>
    </dependency>
    <application xmlns="urn:schemas-microsoft-com:asm.v3">
        <windowsSettings>
            <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2, PerMonitor</dpiAwareness>
            <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">True</dpiAware>
        </windowsSettings>
    </application>
</assembly>
```

prepare an icon file named vol.ico
```
rsrc -manifest epub-comic-repacker.manifest -ico vol.ico -o rsrc.syso
go build
```

#### in MacOS
prepare a PNG file in 1024p, and use the link file downside to build automatically

https://gist.github.com/mholt/11008646c95d787c30806d3f24b2c844
