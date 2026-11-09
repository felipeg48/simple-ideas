// Decodes every .png in a folder with Apple's Vision framework and prints
// "name<TAB>payload" per file (empty payload: not read). Used by TestVision.
import AppKit
import Vision

let dir = CommandLine.arguments[1]
let files = try FileManager.default.contentsOfDirectory(atPath: dir).filter { $0.hasSuffix(".png") }.sorted()
for f in files {
    let path = dir + "/" + f
    var text = ""
    if let img = NSImage(contentsOfFile: path) {
        var r = NSRect(origin: .zero, size: img.size)
        if let cg = img.cgImage(forProposedRect: &r, context: nil, hints: nil) {
            let req = VNDetectBarcodesRequest()
            req.symbologies = [.qr]
            try? VNImageRequestHandler(cgImage: cg).perform([req])
            text = (req.results ?? []).compactMap { $0.payloadStringValue }.first ?? ""
        }
    }
    print(String(f.dropLast(4)) + "\t" + text)
}
