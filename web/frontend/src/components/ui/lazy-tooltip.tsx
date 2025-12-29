import * as React from "react"
import { Tooltip, TooltipTrigger, TooltipContent } from "./tooltip"

interface LazyTooltipProps {
  children: React.ReactNode
  content: React.ReactNode
  side?: "top" | "right" | "bottom" | "left"
  align?: "start" | "center" | "end"
  delayDuration?: number
}

export function LazyTooltip({ 
  children, 
  content, 
  side = "top", 
  align = "center",
  delayDuration = 200
}: LazyTooltipProps) {
  const [mounted, setMounted] = React.useState(false)

  const handleMouseEnter = React.useCallback(() => {
    setMounted(true)
  }, [])

  if (!mounted) {
    return (
      <div onMouseEnter={handleMouseEnter} style={{ display: "contents" }}>
        {children}
      </div>
    )
  }

  return (
    <Tooltip delayDuration={delayDuration} defaultOpen={true}>
      <TooltipTrigger asChild>
        {children}
      </TooltipTrigger>
      <TooltipContent side={side} align={align}>
        {content}
      </TooltipContent>
    </Tooltip>
  )
}
