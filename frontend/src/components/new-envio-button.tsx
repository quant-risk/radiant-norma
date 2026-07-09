'use client'

import { useState } from 'react'
import { Upload } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { UploadModal } from '@/components/upload-modal'

interface NewEnvioButtonProps {
  cadocs: string[]
  token: string
}

export function NewEnvioButton({ cadocs, token }: NewEnvioButtonProps) {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <>
      <Button 
        variant="primary" 
        size="sm" 
        leftIcon={<Upload className="size-3.5" />}
        onClick={() => setIsOpen(true)}
      >
        Novo envio
      </Button>
      <UploadModal
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        cadocs={cadocs}
        onSuccess={() => window.location.reload()}
        token={token}
      />
    </>
  )
}
