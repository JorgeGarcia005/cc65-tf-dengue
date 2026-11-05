# Deja listos para entregar los .docx que genera cmd/word: abre cada uno con
# Microsoft Word, actualiza el índice (números de página reales), lo guarda y
# quita el ajuste "actualizar campos al abrir", para que el profesor no vea
# ese aviso. Con -Pdf exporta además un PDF para revisarlo.
#
# Uso (PowerShell, con Word instalado):
#   go run ./cmd/word
#   powershell -ExecutionPolicy Bypass -File scripts/word-final.ps1 [-Pdf]
param(
    [string]$Carpeta = "entregables",
    [switch]$Pdf
)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem

$archivos = Get-ChildItem -Path $Carpeta -Filter '1ACC0065_*_Informe_*.docx'
if (-not $archivos) { throw "No hay informes 1ACC0065_*_Informe_*.docx en $Carpeta (corre go run ./cmd/word)" }

$word = New-Object -ComObject Word.Application
$word.Visible = $false
$word.DisplayAlerts = 0
try {
    foreach ($a in $archivos) {
        $doc = $word.Documents.Open($a.FullName, $false, $false)
        foreach ($toc in $doc.TablesOfContents) { $toc.Update() }
        $paginas = $doc.ComputeStatistics(2)
        $doc.Save()
        if ($Pdf) { $doc.ExportAsFixedFormat([IO.Path]::ChangeExtension($a.FullName, '.pdf'), 17) }
        $doc.Close(0)
        "$($a.Name): $paginas páginas"
    }
} finally {
    $word.Quit()
    [void][Runtime.InteropServices.Marshal]::ReleaseComObject($word)
}

# Quitar <w:updateFields> de word/settings.xml (el índice ya está actualizado).
foreach ($a in $archivos) {
    $zip = [IO.Compression.ZipFile]::Open($a.FullName, 'Update')
    try {
        $entrada = $zip.GetEntry('word/settings.xml')
        $lector = New-Object IO.StreamReader($entrada.Open())
        $xml = $lector.ReadToEnd()
        $lector.Close()
        $nuevo = $xml -replace '<w:updateFields[^>]*/>', ''
        if ($nuevo -ne $xml) {
            $entrada.Delete()
            $escritor = New-Object IO.StreamWriter($zip.CreateEntry('word/settings.xml').Open(), (New-Object Text.UTF8Encoding $false))
            $escritor.Write($nuevo)
            $escritor.Close()
        }
    } finally {
        $zip.Dispose()
    }
}
"listo: $($archivos.Count) informe(s) en $Carpeta"
