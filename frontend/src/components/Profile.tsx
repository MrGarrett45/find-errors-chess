import { useAuth0 } from '@auth0/auth0-react'

export function Profile() {
  const { user, isAuthenticated, isLoading } = useAuth0()

  if (isLoading || !isAuthenticated || !user) {
    return null
  }

  return (
    <div className="profile-pill">
      <img
        src={user.picture ?? '/theory-gap-logo.png'}
        alt={user.name ?? 'User avatar'}
        className="profile-avatar"
        onError={(event) => {
          const target = event.currentTarget
          target.src = '/theory-gap-logo.png'
        }}
      />
      <div className="profile-text">
        <div className="profile-name">{user.name ?? 'Authenticated'}</div>
        {user.email ? <div className="profile-email">{user.email}</div> : null}
      </div>
    </div>
  )
}
