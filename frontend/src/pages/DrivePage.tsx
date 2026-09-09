import { useEffect, useState } from 'react'
import { FileListPanel } from '../components/files/FileListPanel'
import { UploadDropzone } from '../components/files/UploadDropzone'
import { useBreadcrumbs } from '../contexts/BreadcrumbContext'

export function DrivePage() {
  const { setBreadcrumbs } = useBreadcrumbs()
  const [refreshVersion, setRefreshVersion] = useState(0)

  useEffect(() => {
    setBreadcrumbs([{ label: 'My Drive', href: '/' }])
  }, [setBreadcrumbs])

  return (
    <div className="space-y-6 p-6 sm:p-8">
      <UploadDropzone onUploaded={() => setRefreshVersion((v) => v + 1)} />
      <FileListPanel refreshVersion={refreshVersion} />
    </div>
  )
}
