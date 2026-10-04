import { useState, type ComponentType, type ReactNode } from 'react'
import {
  CircleUserRound,
  Image,
  ScanSearch,
  Store,
  UserRoundCheck,
  UserRoundSearch,
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ActionCard } from '@/components/shared/action-card'
import { PageHeader } from '@/components/shared/page-header'
import { DeviceGuard, useSelectedDevice } from '@/hooks/use-device-guard'
import { cn } from '@/lib/utils'
import { RecipientBar } from '@/features/messaging/recipient-bar'
import { AvatarForm } from '@/features/account/avatar-form'
import { BusinessProfileForm } from '@/features/account/business-profile-form'
import { ChangeAvatarForm } from '@/features/account/change-avatar-form'
import { ContactsView } from '@/features/account/contacts-view'
import { PrivacyView } from '@/features/account/privacy-view'
import { MyProfileCard } from '@/features/account/profile-card'
import { PushnameForm } from '@/features/account/pushname-form'
import { UserCheckForm } from '@/features/account/user-check-form'
import { UserInfoForm } from '@/features/account/user-info-form'

interface LookupType {
  value: string
  label: string
  description: string
  icon: ComponentType<{ className?: string }>
  form: ReactNode
}

const lookups: LookupType[] = [
  {
    value: 'info',
    label: 'User info',
    description: "Look up a user's public info by phone or LID.",
    icon: UserRoundSearch,
    form: <UserInfoForm />,
  },
  {
    value: 'check',
    label: 'User check',
    description: 'Check whether a number is on WhatsApp.',
    icon: UserRoundCheck,
    form: <UserCheckForm />,
  },
  {
    value: 'avatar',
    label: 'Avatar',
    description: "Fetch a user's profile picture by phone or LID.",
    icon: Image,
    form: <AvatarForm />,
  },
  {
    value: 'business',
    label: 'Business profile',
    description: "Look up a business user's profile, catalog, and category.",
    icon: Store,
    form: <BusinessProfileForm />,
  },
]

function LookupsPanel() {
  const [type, setType] = useState('info')
  const active = lookups.find((item) => item.value === type) ?? lookups[0]

  return (
    <div className="grid gap-3 sm:gap-4 lg:grid-cols-[210px_1fr]">
      {/* Lookup picker: vertical glass segment on desktop */}
      <div className="glass-card hidden flex-col gap-1 rounded-xl p-2 backdrop-blur-xl lg:flex">
        <p className="px-2.5 pb-1 text-[10px] font-semibold tracking-wider text-muted-foreground/70 uppercase">
          Directory Query
        </p>
        {lookups.map(({ value, label, icon: Icon }) => (
          <button
            key={value}
            type="button"
            onClick={() => setType(value)}
            aria-pressed={type === value}
            className={cn(
              'flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-left text-xs font-medium transition-all duration-150',
              type === value
                ? 'border border-primary/25 bg-primary/12 font-semibold text-primary shadow-2xs'
                : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
            )}
          >
            <Icon className="size-3.5" />
            <span>{label}</span>
          </button>
        ))}
      </div>

      {/* Lookup picker: dropdown on mobile */}
      <div className="flex flex-col gap-1.5 lg:hidden">
        <Label className="text-xs">Lookup Query Type</Label>
        <Select value={type} onValueChange={setType}>
          <SelectTrigger className="h-8.5 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent className="text-xs">
            {lookups.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card className="glass-card rounded-xl backdrop-blur-xl">
        <div key={active.value} className="animate-in fade-in flex flex-col gap-3 p-4 sm:p-5 duration-200">
          <CardHeader className="p-0 pb-3 border-b border-border/50">
            <CardTitle className="text-sm font-semibold">{active.label}</CardTitle>
            <CardDescription className="text-xs">{active.description}</CardDescription>
          </CardHeader>
          <CardContent className="p-0 pt-2">{active.form}</CardContent>
        </div>
      </Card>
    </div>
  )
}

export default function AccountPage() {
  const device = useSelectedDevice()

  return (
    <div className="flex flex-col gap-4 sm:gap-5">
      <PageHeader
        title="Account & Identity"
        description="Active device profile settings, contact directories, and privacy configuration."
      />
      {device === null ? (
        <DeviceGuard />
      ) : (
        <>
          <RecipientBar />
          <Tabs defaultValue="profile" className="gap-3">
            <TabsList className="h-9 rounded-lg border border-border/70 bg-card/60 p-1 backdrop-blur-md">
              <TabsTrigger value="profile" className="h-7 rounded-md text-xs font-medium">
                My Profile
              </TabsTrigger>
              <TabsTrigger value="lookups" className="h-7 rounded-md text-xs font-medium">
                <ScanSearch className="size-3.5" />
                Directory Search
              </TabsTrigger>
              <TabsTrigger value="contacts" className="h-7 rounded-md text-xs font-medium">
                Synced Contacts
              </TabsTrigger>
              <TabsTrigger value="privacy" className="h-7 rounded-md text-xs font-medium">
                Privacy Settings
              </TabsTrigger>
            </TabsList>
            <TabsContent value="profile" className="flex flex-col gap-4 pt-1">
              <MyProfileCard />
              <div className="grid gap-3 sm:grid-cols-2">
                <ActionCard
                  icon={CircleUserRound}
                  title="Push Name"
                  description="The display name other WhatsApp users see for you."
                >
                  <PushnameForm />
                </ActionCard>
                <ActionCard
                  icon={Image}
                  title="Profile Picture"
                  description="Upload a new avatar or remove your current one."
                >
                  <ChangeAvatarForm />
                </ActionCard>
              </div>
            </TabsContent>
            <TabsContent value="lookups" className="pt-1">
              <LookupsPanel />
            </TabsContent>
            <TabsContent value="contacts" className="pt-1">
              <ContactsView />
            </TabsContent>
            <TabsContent value="privacy" className="pt-1">
              <PrivacyView />
            </TabsContent>
          </Tabs>
        </>
      )}
    </div>
  )
}
