"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldLabel, FieldError } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Progress } from "@/components/ui/progress"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { authErrorMessage } from "@/features/auth/hooks"
import { useCompleteOnboarding, useOnboardingStatus } from "@/features/onboarding/hooks"
import { LogoPicker } from "@/features/onboarding/components/logo-picker"
import { OnboardingRequestAccountType } from "@/lib/api/generated/model"

const INDUSTRIES = [
  "Construction",
  "Information Technology",
  "Healthcare",
  "Manufacturing",
  "Logistics & Transport",
  "Professional Services",
  "Retail & Wholesale",
  "Education",
  "Other",
] as const

export const COMPANY_SIZES = ["1-10", "11-50", "51-200", "201-500", "500+"] as const
export const CONTRACT_SIZES = ["<$50K", "$50K-$200K", "$200K-$1M", ">$1M"] as const

const OnboardingSchema = z
  .object({
    accountType: z.enum(["buyer", "supplier", "both"]),
    orgName: z.string().min(1, "Organisation name is required."),
    businessName: z.string().optional(),
    businessDescription: z.string().optional(),
    industry: z.string().min(1, "Industry is required."),
    industryOther: z.string().optional(),
    companySize: z.string().optional(),
    geographicCoverage: z.string().optional(),
    preferredContractSize: z.string().optional(),
  })
  .refine((v) => v.industry !== "Other" || !!v.industryOther?.trim(), {
    path: ["industryOther"],
    message: "Please specify your industry.",
  })

type OnboardingValues = z.infer<typeof OnboardingSchema>

const STEPS = [
  {
    title: "What best describes you?",
    description: "We'll tailor tenders to your role.",
    fields: ["accountType"] as const,
  },
  {
    title: "Tell us about your organisation",
    description: "This helps us tailor tenders to your business.",
    fields: [
      "orgName",
      "businessName",
      "businessDescription",
      "companySize",
      "geographicCoverage",
      "preferredContractSize",
    ] as const,
  },
  {
    title: "What industry are you in?",
    description: "Almost done.",
    fields: ["industry", "industryOther"] as const,
  },
  {
    title: "Add your logo",
    description: "Optional — you can always add this later in settings.",
    fields: [] as const,
  },
]

export function OnboardingForm() {
  const router = useRouter()
  const completeOnboarding = useCompleteOnboarding()
  const { data: onboarding } = useOnboardingStatus()
  const [showOtherIndustry, setShowOtherIndustry] = useState(false)
  const [step, setStep] = useState(0)

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    trigger,
    formState: { errors },
  } = useForm<OnboardingValues>({
    resolver: zodResolver(OnboardingSchema),
    defaultValues: {
      accountType: "buyer",
      orgName: "",
      businessName: "",
      businessDescription: "",
      industry: "",
      industryOther: "",
      companySize: "",
      geographicCoverage: "",
      preferredContractSize: "",
    },
  })

  const accountType = watch("accountType")
  const industry = watch("industry")
  const companySize = watch("companySize")
  const preferredContractSize = watch("preferredContractSize")
  const isLogoStep = step === STEPS.length - 1
  const isDataStep = step === STEPS.length - 2

  function onSubmit(values: OnboardingValues) {
    completeOnboarding.mutate(
      {
        accountType: values.accountType as OnboardingRequestAccountType,
        orgName: values.orgName,
        businessName: values.businessName?.trim() ? values.businessName.trim() : undefined,
        businessDescription: values.businessDescription?.trim()
          ? values.businessDescription.trim()
          : undefined,
        industry: values.industry === "Other" ? values.industryOther!.trim() : values.industry,
        companySize: values.companySize || undefined,
        geographicCoverage: values.geographicCoverage?.trim() || undefined,
        preferredContractSize: values.preferredContractSize || undefined,
      },
      {
        onSuccess: () => setStep((s) => s + 1),
        onError: (err) => toast.error(authErrorMessage(err)),
      }
    )
  }

  async function onContinue() {
    const valid = await trigger(STEPS[step].fields)
    if (valid) setStep((s) => s + 1)
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <Progress value={((step + 1) / STEPS.length) * 100} className="mb-2" />
        <CardTitle>{STEPS[step].title}</CardTitle>
        <CardDescription>{STEPS[step].description}</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          onSubmit={isDataStep ? handleSubmit(onSubmit) : (e) => e.preventDefault()}
          className="flex flex-col gap-4"
        >
          {step === 0 && (
            <Field>
              <FieldLabel>I am a</FieldLabel>
              <div className="grid grid-cols-3 gap-2">
                {(["buyer", "supplier", "both"] as const).map((type) => (
                  <Button
                    key={type}
                    type="button"
                    variant={accountType === type ? "default" : "outline"}
                    onClick={() => setValue("accountType", type)}
                  >
                    {type === "buyer" ? "Buyer" : type === "supplier" ? "Supplier" : "Both"}
                  </Button>
                ))}
              </div>
              {errors.accountType && <FieldError>{errors.accountType.message}</FieldError>}
            </Field>
          )}

          {step === 1 && (
            <>
              <Field>
                <FieldLabel htmlFor="orgName">Organisation name</FieldLabel>
                <Input id="orgName" {...register("orgName")} />
                {errors.orgName && <FieldError>{errors.orgName.message}</FieldError>}
              </Field>

              <Field>
                <FieldLabel htmlFor="businessName">Business name (optional)</FieldLabel>
                <Input id="businessName" {...register("businessName")} />
              </Field>

              <Field>
                <FieldLabel htmlFor="businessDescription">
                  Business description (optional)
                </FieldLabel>
                <Textarea id="businessDescription" {...register("businessDescription")} />
              </Field>

              <Field>
                <FieldLabel htmlFor="companySize">Company size (optional)</FieldLabel>
                <Select
                  value={companySize}
                  onValueChange={(v) => setValue("companySize", v)}
                >
                  <SelectTrigger id="companySize" className="w-full">
                    <SelectValue placeholder="Select company size" />
                  </SelectTrigger>
                  <SelectContent>
                    {COMPANY_SIZES.map((option) => (
                      <SelectItem key={option} value={option}>
                        {option} employees
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel htmlFor="geographicCoverage">
                  Geographic coverage (optional)
                </FieldLabel>
                <Input
                  id="geographicCoverage"
                  placeholder="e.g. Singapore-wide"
                  {...register("geographicCoverage")}
                />
              </Field>

              <Field>
                <FieldLabel htmlFor="preferredContractSize">
                  Preferred contract size (optional)
                </FieldLabel>
                <Select
                  value={preferredContractSize}
                  onValueChange={(v) => setValue("preferredContractSize", v)}
                >
                  <SelectTrigger id="preferredContractSize" className="w-full">
                    <SelectValue placeholder="Select preferred contract size" />
                  </SelectTrigger>
                  <SelectContent>
                    {CONTRACT_SIZES.map((option) => (
                      <SelectItem key={option} value={option}>
                        {option}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            </>
          )}

          {step === 2 && (
            <>
              <Field>
                <FieldLabel htmlFor="industry">Industry</FieldLabel>
                <Select
                  value={industry}
                  onValueChange={(v) => {
                    setValue("industry", v)
                    setShowOtherIndustry(v === "Other")
                  }}
                >
                  <SelectTrigger id="industry" className="w-full">
                    <SelectValue placeholder="Select industry" />
                  </SelectTrigger>
                  <SelectContent>
                    {INDUSTRIES.map((option) => (
                      <SelectItem key={option} value={option}>
                        {option}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {errors.industry && <FieldError>{errors.industry.message}</FieldError>}
              </Field>

              {showOtherIndustry && (
                <Field>
                  <FieldLabel htmlFor="industryOther">Please specify</FieldLabel>
                  <Input id="industryOther" {...register("industryOther")} />
                  {errors.industryOther && <FieldError>{errors.industryOther.message}</FieldError>}
                </Field>
              )}
            </>
          )}

          {isLogoStep && <LogoPicker logoUrl={onboarding?.logoUrl} />}

          <div className="flex gap-2">
            {step > 0 && !isLogoStep && (
              <Button type="button" variant="outline" onClick={() => setStep((s) => s - 1)}>
                Back
              </Button>
            )}
            {isLogoStep ? (
              <Button type="button" className="flex-1" onClick={() => router.replace("/")}>
                Finish
              </Button>
            ) : isDataStep ? (
              <Button type="submit" disabled={completeOnboarding.isPending} className="flex-1">
                {completeOnboarding.isPending ? "Saving…" : "Continue"}
              </Button>
            ) : (
              <Button type="button" onClick={onContinue} className="flex-1">
                Continue
              </Button>
            )}
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
