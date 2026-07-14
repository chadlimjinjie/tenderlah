"use client"

import * as React from "react"
import { toast } from "sonner"
import { ImagePlusIcon } from "lucide-react"

import { authErrorMessage } from "@/features/auth/hooks"
import { useUploadOnboardingLogo } from "@/features/onboarding/hooks"

/** Single-image picker for the business logo: click to upload/replace. */
export function LogoPicker({ logoUrl }: { logoUrl?: string | null }) {
  const fileInputRef = React.useRef<HTMLInputElement>(null)
  const upload = useUploadOnboardingLogo()

  function handleFile(files: FileList | null) {
    const file = files?.[0]
    if (!file) return
    upload.mutate(file, { onError: (err) => toast.error(authErrorMessage(err)) })
    if (fileInputRef.current) fileInputRef.current.value = ""
  }

  return (
    <div className="flex items-center gap-3">
      <button
        type="button"
        onClick={() => fileInputRef.current?.click()}
        disabled={upload.isPending}
        className="relative flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-md border border-dashed text-muted-foreground hover:bg-accent"
      >
        {logoUrl ? (
          <img src={logoUrl} alt="Business logo" className="h-full w-full object-cover" />
        ) : (
          <ImagePlusIcon className="h-5 w-5" />
        )}
      </button>
      <div className="text-sm text-muted-foreground">
        {upload.isPending ? "Uploading…" : logoUrl ? "Click to replace logo" : "Click to add a logo (optional)"}
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={(e) => handleFile(e.target.files)}
      />
    </div>
  )
}
